package sqlstorage

import "go.uber.org/zap"

type Option func(opt *options)

type options struct {
	logger     *zap.Logger
	sqlUrl     string
	BatchCount int
}

var defaultOptions = options{
	logger: zap.NewNop(),
}

func WithLogger(logger *zap.Logger) Option {
	return func(opt *options) {
		opt.logger = logger
	}
}

func WithSQLURL(sqlUrl string) Option {
	return func(opt *options) {
		opt.sqlUrl = sqlUrl
	}
}

func WithBatchCount(batchCount int) Option {
	return func(opt *options) {
		opt.BatchCount = batchCount
	}
}
