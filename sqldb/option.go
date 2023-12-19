package sqldb

import "go.uber.org/zap"

type options struct {
	logger *zap.Logger
	sqlUrl string
}

var defaultOption = options{
	logger: zap.NewNop(),
}

type Option func(opt *options)

func WithLogger(logger *zap.Logger) Option {
	return func(opt *options) {
		opt.logger = logger
	}
}

func WithSqlUrl(sqlUrl string) Option {
	return func(opt *options) {
		opt.sqlUrl = sqlUrl
	}
}
