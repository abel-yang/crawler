package collect

import (
	"errors"
	"time"
)

type ParseResult struct {
	Requests []*Request    // 网站获取到的资源链接
	Items    []interface{} //网站获取到的数据
}

type Request struct {
	Url       string
	WaitTime  time.Duration
	Depth     int
	MaxDepth  int
	Cookie    string
	ParseFunc func([]byte, *Request) ParseResult
}

func (r *Request) Check() error {
	if r.Depth > r.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}
