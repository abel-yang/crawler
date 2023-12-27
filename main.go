package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/abel-yang/crawler/collect"
	"github.com/abel-yang/crawler/engine"
	"github.com/abel-yang/crawler/limiter"
	"github.com/abel-yang/crawler/log"
	pb "github.com/abel-yang/crawler/proto/greeter"
	"github.com/abel-yang/crawler/proxy"
	"github.com/abel-yang/crawler/spider"
	"github.com/abel-yang/crawler/storage/sqlstorage"
	"github.com/go-micro/plugins/v4/config/encoder/toml"
	"github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/config"
	"go-micro.dev/v4/config/reader"
	"go-micro.dev/v4/config/reader/json"
	"go-micro.dev/v4/config/source"
	"go-micro.dev/v4/config/source/file"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"os"
	"time"
)

// Version information.
var (
	BuildTS   = "None"
	GitHash   = "None"
	GitBranch = "None"
	Version   = "None"
)

func GetVersion() string {
	if GitHash != "" {
		h := GitHash
		if len(h) > 7 {
			h = h[:7]
		}
		return fmt.Sprintf("%s-%s", Version, h)
	}
	return Version
}

// Printer print build version
func Printer() {
	fmt.Println("Version:          ", GetVersion())
	fmt.Println("Git Branch:       ", GitBranch)
	fmt.Println("Git Commit:       ", GitHash)
	fmt.Println("Build Time (UTC): ", BuildTS)
}

var (
	PrintVersion = flag.Bool("version", false, "print the version of this build")
)

func main() {
	flag.Parse()
	if *PrintVersion {
		Printer()
		os.Exit(0)
	}
	enc := toml.NewEncoder()
	cfg, err := config.NewConfig(config.WithReader(json.NewReader(reader.WithEncoder(enc))))
	err = cfg.Load(file.NewSource(
		file.WithPath("config.toml"),
		source.WithEncoder(enc),
	))

	if err != nil {
		panic(err)
	}

	logLevelText := cfg.Get("logLevel").String("INFO")
	logLevel, err := zapcore.ParseLevel(logLevelText)
	if err != nil {
		panic(err)
	}
	plugin := log.NewStdoutPlugin(logLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	//set zap global logger
	zap.ReplaceGlobals(logger)

	startCrawler(cfg, logger)

	var serverCfg ServerConfig
	if err := cfg.Get("GRPCServer").Scan(&serverCfg); err != nil {
		logger.Error("get GRPC Server config failed", zap.Error(err))
	}
	logger.Sugar().Debugf("grpc server config,%+v", serverCfg)

	// start http proxy to GRPC
	go RunHTTPServer(serverCfg)

	RunGRPCServer(logger, serverCfg)
}

func RunGRPCServer(logger *zap.Logger, serverCfg ServerConfig) {
	reg := etcd.NewRegistry(
		registry.Addrs(serverCfg.RegistryAddress),
	)

	//生成grpc server
	service := micro.NewService(
		micro.Server(gs.NewServer(server.Id(serverCfg.ID))),
		micro.Address(serverCfg.GRPCListenAddress),
		micro.Registry(reg),
		micro.RegisterTTL(time.Duration(serverCfg.RegisterTTL)*time.Second),
		micro.RegisterInterval(time.Duration(server.DefaultRegisterInterval)*time.Second),
		micro.Name("go.micro.server.worker"),
		micro.WrapHandler(logWrapper(logger)),
	)

	// 设置micro 客户端默认超时时间为10秒钟
	if err := service.Client().Init(client.RequestTimeout(time.Duration(serverCfg.ClientTimeOut) * time.Second)); err != nil {
		logger.Sugar().Error("micro client init error. ", zap.String("error:", err.Error()))

		return
	}

	// parse command line flags
	service.Init()

	if err := pb.RegisterGreeterHandler(service.Server(), new(Greeter)); err != nil {
		logger.Fatal("register handler failed")

		return
	}
	if err := service.Run(); err != nil {
		logger.Fatal("grpc mirco failed", zap.Error(err))
	}
}

func RunHTTPServer(cfg ServerConfig) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, cfg.GRPCListenAddress, opts); err != nil {
		zap.L().Fatal("Register backend grpc server endpoint failed")
	}
	zap.S().Debugf("start http server listening on %v proxy to grpc server;%v", cfg.HTTPListenAddress, cfg.GRPCListenAddress)
	if err := http.ListenAndServe(cfg.HTTPListenAddress, mux); err != nil {
		zap.L().Fatal("http listenAndServe failed")

		return
	}
}

type ServerConfig struct {
	GRPCListenAddress string
	HTTPListenAddress string
	ID                string
	RegistryAddress   string
	RegisterTTL       int
	RegisterInterval  int
	Name              string
	ClientTimeOut     int
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) (err error) {
	rsp.Greeting = "hello " + req.Name
	return nil
}

func startCrawler(cfg config.Config, logger *zap.Logger) {
	proxyUrl := cfg.Get("fetcher", "proxy").StringSlice([]string{})
	timeout := cfg.Get("fetcher", "timeout").Int(5000)
	p, err := proxy.RoundRobinProxySwitcher(proxyUrl)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed")

		return
	}
	var f spider.Fetcher = collect.BrowserFetch{
		Timeout: time.Duration(timeout) * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}

	var storage spider.Storage
	sqlURL := cfg.Get("storage", "sqlURL").String("")
	if storage, err = sqlstorage.New(
		sqlstorage.WithSQLURL(sqlURL),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(2),
	); err != nil {
		logger.Error("create sqlStorage failed", zap.Error(err))
		return
	}

	//init task
	var taskCfg []spider.TaskConfig
	if err := cfg.Get("Tasks").Scan(&taskCfg); err != nil {
		logger.Error("init seed tasks", zap.Error(err))
		return
	}

	seeds := ParseTaskConfig(logger, f, storage, taskCfg)
	s := engine.NewEngine(
		engine.WithFetcher(f),
		engine.WithWorkCount(5),
		engine.WithLogger(logger),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)

	go s.Run()
}

func ParseTaskConfig(logger *zap.Logger, f spider.Fetcher, storage spider.Storage, cfgs []spider.TaskConfig) []*spider.Task {
	tasks := make([]*spider.Task, 0, 1000)
	for _, cfg := range cfgs {
		t := spider.NewTask(
			spider.WithName(cfg.Name),
			spider.WithReload(cfg.Reload),
			spider.WithCookie(cfg.Cookie),
			spider.WithLogger(logger),
			spider.WithStorage(storage),
		)

		if cfg.WaitTime > 0 {
			t.WaitTime = cfg.WaitTime
		}

		if cfg.MaxDepth > 0 {
			t.MaxDepth = cfg.MaxDepth
		}

		var limits []limiter.RateLimiter
		if len(limits) > 0 {
			for _, lcfg := range cfg.Limits {
				l := rate.NewLimiter(limiter.Per(lcfg.EventCount, time.Duration(lcfg.EventDur)*time.Second), 1)
				limits = append(limits, l)
			}
			multiLimit := limiter.MultiLimiter(limits...)
			t.Limit = multiLimit
		}

		switch cfg.Fetcher {
		case "browser":
			t.Fetcher = f
		}
		tasks = append(tasks, t)
	}

	return tasks
}

func logWrapper(log *zap.Logger) server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			log.Info("receive request:",
				zap.String("method", req.Method()),
				zap.String("service", req.Service()),
				zap.Reflect("request params", req.Body()))
			return fn(ctx, req, rsp)
		}
	}
}

func loadConfig() {
	err := config.Load(file.NewSource(
		file.WithPath("config.json"),
	))
	if err != nil {
		fmt.Println(err)
	}
	type Host struct {
		Address string `json:"address"`
		Port    int    `json:"port"`
	}
	var host Host
	// 获取hosts.database下的数据，并解析为host结构
	config.Get("hosts", "database").Scan(&host)

	fmt.Println(host)

	w, err := config.Watch("hosts", "database")
	if err != nil {
		fmt.Println(err)
	}

	//等待配置文件更新
	v, err := w.Next()
	if err != nil {
		fmt.Println(err)
	}

	v.Scan(&host)
	fmt.Println(host)
}
