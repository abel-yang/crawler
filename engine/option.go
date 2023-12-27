package engine

import (
	"github.com/abel-yang/crawler/spider"
	"go.uber.org/zap"
)

type Option func(opt *options)

type options struct {
	WorkCount int
	Logger    *zap.Logger
	Fetcher   spider.Fetcher
	Seeds     []*spider.Task
	scheduler Scheduler
}

var defaultOptions = options{
	Logger: zap.NewNop(),
}

func WithLogger(logger *zap.Logger) Option {
	return func(opt *options) {
		opt.Logger = logger
	}
}

func WithFetcher(fetch spider.Fetcher) Option {
	return func(opt *options) {
		opt.Fetcher = fetch
	}
}

func WithWorkCount(workCount int) Option {
	return func(opt *options) {
		opt.WorkCount = workCount
	}
}

func WithSeeds(seeds []*spider.Task) Option {
	return func(opt *options) {
		opt.Seeds = seeds
	}
}

func WithScheduler(scheduler Scheduler) Option {
	return func(opt *options) {
		opt.scheduler = scheduler
	}
}
