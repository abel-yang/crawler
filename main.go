package main

import (
	"github.com/abel-yang/crawler/collect"
	"github.com/abel-yang/crawler/engine"
	"github.com/abel-yang/crawler/log"
	"github.com/abel-yang/crawler/proxy"
	"go.uber.org/zap/zapcore"
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

	var seeds = make([]*collect.Task, 0, 1000)
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "js_find_douban_sun_room",
		},
		Fetcher: f,
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
