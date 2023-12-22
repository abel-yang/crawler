package main

import (
	"context"
	"fmt"
	"github.com/abel-yang/crawler/collect"
	"github.com/abel-yang/crawler/collector"
	"github.com/abel-yang/crawler/collector/sqlstorage"
	"github.com/abel-yang/crawler/engine"
	"github.com/abel-yang/crawler/limiter"
	"github.com/abel-yang/crawler/log"
	pb "github.com/abel-yang/crawler/proto/greeter"
	"github.com/abel-yang/crawler/proxy"
	"github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"net/http"
	"time"
)

func main() {
	plugin := log.NewStdoutPlugin(zapcore.InfoLevel)
	logger := log.NewLogger(plugin)

	go HandleHTTP()

	reg := etcd.NewRegistry(
		registry.Addrs(":2379"),
	)

	//生成grpc server
	service := micro.NewService(
		micro.Server(gs.NewServer()),
		micro.Address(":9000"),
		micro.Name("go.micro.server.worker"),
		micro.Registry(reg),
	)

	// parse command line flags
	service.Init()
	pb.RegisterGreeterHandler(service.Server(), new(Greeter))
	if err := service.Run(); err != nil {
		logger.Fatal("grpc mirco failed", zap.Error(err))
	}
}

func HandleHTTP() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, "localhost:9000", opts)
	if err != nil {
		fmt.Println(err)
	}
	http.ListenAndServe(":8080", mux)
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) (err error) {
	rsp.Greeting = "hello " + req.Name
	return nil
}

func startCrawler() {
	plugin := log.NewStdoutPlugin(zapcore.InfoLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	proxyUrl := []string{"http://127.0.0.1:9981", "http://127.0.0.1:9981"}
	p, err := proxy.RoundRobinProxySwitcher(proxyUrl)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed")
		return
	}
	var f collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}

	var storage collector.Storage
	storage, err = sqlstorage.New(
		sqlstorage.WithSqlUrl("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(2),
	)
	if err != nil {
		logger.Error("create sqlstorage failed", zap.Error(err))
		return
	}
	//2秒钟1个
	secondLimit := rate.NewLimiter(Per(1, 2*time.Second), 1)
	//60秒20个
	minuteLimit := rate.NewLimiter(Per(20, 60*time.Second), 20)
	multiLimit := limiter.MultiLimiter(secondLimit, minuteLimit)

	var seeds = make([]*collect.Task, 0, 1000)
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "douban_book_list",
		},
		Fetcher: f,
		Storage: storage,
		Limit:   multiLimit,
	})

	s := engine.NewEngine(
		engine.WithFetcher(f),
		engine.WithWorkCount(5),
		engine.WithLogger(logger),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)

	s.Run()
}

func Per(everyCount int, duration time.Duration) rate.Limit {
	return rate.Every(duration / time.Duration(everyCount))
}
