package engine

import (
	"github.com/abel-yang/crawler/collect"
	"go.uber.org/zap"
)

type Option func(opt *options)

type options struct {
	WorkCount int
	Logger    *zap.Logger
	Fetcher   collect.Fetcher
	Seeds     []*collect.Request
}

var defaultOptions = options{
	Logger: zap.NewNop(),
}

func WithLogger(logger *zap.Logger) Option {
	return func(opt *options) {
		opt.Logger = logger
	}
}

func WithFetcher(fetch collect.Fetcher) Option {
	return func(opt *options) {
		opt.Fetcher = fetch
	}
}

func WithWorkCount(workCount int) Option {
	return func(opt *options) {
		opt.WorkCount = workCount
	}
}

func WithSeeds(seeds []*collect.Request) Option {
	return func(opt *options) {
		opt.Seeds = seeds
	}
}
