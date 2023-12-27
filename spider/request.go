package spider

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"math/rand"
	"regexp"
	"time"
)

type ParseResult struct {
	Requests []*Request    // 网站获取到的资源链接
	Items    []interface{} //网站获取到的数据
}

type Context struct {
	Body []byte
	Req  *Request
}

// 单个请求
type Request struct {
	unique   string
	Task     *Task
	Priority int
	Url      string
	Depth    int
	Method   string
	RuleName string
	TmpData  *Temp
}

var ErrMaxDepthLimitReached = errors.New("max depth limit reached")

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return ErrMaxDepthLimitReached
	}
	return nil
}

func (r *Request) Fetch() ([]byte, error) {
	if err := r.Task.Limit.Wait(context.Background()); err != nil {
		return nil, err
	}
	//随机休眠，模拟人行为
	sleepTime := rand.Int63n(r.Task.WaitTime * 1000)
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	return r.Task.Fetcher.Get(r)
}

// Unique 请求唯一标识
func (r *Request) Unique() string {
	identify := md5.Sum([]byte(r.Url + r.Method))

	return hex.EncodeToString(identify[:])
}

func (ctx *Context) ParseJSReg(name string, reg string) ParseResult {
	re := regexp.MustCompile(reg)
	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := ParseResult{}
	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(result.Requests, &Request{
			Method:   "Get",
			Task:     ctx.Req.Task,
			Url:      u,
			Depth:    ctx.Req.Depth + 1,
			RuleName: name,
		})
	}
	return result
}

func (ctx *Context) OutputJS(reg string) ParseResult {
	re := regexp.MustCompile(reg)
	if ok := re.Match(ctx.Body); !ok {
		return ParseResult{
			Items: []interface{}{},
		}
	}
	return ParseResult{
		Items: []interface{}{ctx.Req.Url},
	}
}

func (ctx *Context) Output(data interface{}) *DataCell {
	res := &DataCell{}
	res.Data = make(map[string]interface{})
	res.Data["Task"] = ctx.Req.Task.Name
	res.Data["table"] = ctx.Req.Task.Name
	res.Data["Rule"] = ctx.Req.RuleName
	res.Data["Data"] = data
	res.Data["Url"] = ctx.Req.Url
	res.Data["Time"] = time.Now().Format("2000-01-01 00:00:00")
	return res
}
