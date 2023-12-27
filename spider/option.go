package spider

import (
	"github.com/abel-yang/crawler/limiter"
	"go.uber.org/zap"
)

type Options struct {
	Name     string `json:"name"` // 任务名称，应保证唯一性
	URL      string `json:"url"`
	Cookie   string `json:"cookie"`
	WaitTime int64  `json:"wait_time"` // 随机休眠时间，秒
	Reload   bool   `json:"reload"`    // 网站是否可以重复爬取
	MaxDepth int64  `json:"max_depth"`
	Fetcher  Fetcher
	Storage  Storage
	Limit    limiter.RateLimiter
	Logger   *zap.Logger
}

var defaultOptions = Options{
	Logger:   zap.NewNop(),
	WaitTime: 5,
	Reload:   false,
	MaxDepth: 5,
}

type Option func(opt *Options)

func WithLogger(logger *zap.Logger) Option {
	return func(opt *Options) {
		opt.Logger = logger
	}
}

func WithName(name string) Option {
	return func(opt *Options) {
		opt.Name = name
	}
}

func WithURL(url string) Option {
	return func(opts *Options) {
		opts.URL = url
	}
}

func WithCookie(cookie string) Option {
	return func(opts *Options) {
		opts.Cookie = cookie
	}
}

func WithWaitTime(waitTime int64) Option {
	return func(opts *Options) {
		opts.WaitTime = waitTime
	}
}

func WithReload(reload bool) Option {
	return func(opts *Options) {
		opts.Reload = reload
	}
}

func WithFetcher(f Fetcher) Option {
	return func(opts *Options) {
		opts.Fetcher = f
	}
}

func WithStorage(s Storage) Option {
	return func(opts *Options) {
		opts.Storage = s
	}
}

func WithMaxDepth(maxDepth int64) Option {
	return func(opts *Options) {
		opts.MaxDepth = maxDepth
	}
}
