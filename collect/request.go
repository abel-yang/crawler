package collect

import (
	"errors"
	"time"
)

type ParseResult struct {
	Requests []*Request    // 网站获取到的资源链接
	Items    []interface{} //网站获取到的数据
}

type Task struct {
	Url      string
	WaitTime time.Duration
	MaxDepth int
	Cookie   string
	RootReq  *Request
	Fetcher  Fetcher
}

// 单个请求
type Request struct {
	Task      *Task
	Url       string
	Depth     int
	ParseFunc func([]byte, *Request) ParseResult
}

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}
