package main

import (
	"github.com/abel-yang/crawler/collect"
	"github.com/abel-yang/crawler/collector"
	"github.com/abel-yang/crawler/collector/sqlstorage"
	"github.com/abel-yang/crawler/engine"
	"github.com/abel-yang/crawler/limiter"
	"github.com/abel-yang/crawler/log"
	"github.com/abel-yang/crawler/proxy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"time"
)

func main() {
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
