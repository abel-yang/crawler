package collect

import (
	"crypto/md5"
	"encoding/hex"
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
	Priority  int
	Url       string
	Depth     int
	Method    string
	ParseFunc func([]byte, *Request) ParseResult
}

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("max depth limit reached")
	}
	return nil
}

// Unique 请求唯一标识
func (r *Request) Unique() string {
	identify := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(identify[:])
}
