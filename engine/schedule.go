package engine

import (
	"fmt"
	"github.com/abel-yang/crawler/collect"
	"github.com/abel-yang/crawler/parse/doubanbook"
	"github.com/abel-yang/crawler/parse/doubanggroup"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"sync"
	"time"
)

func init() {
	Store.Add(doubanggroup.DoubangroupTask)
	Store.AddJSTask(doubanggroup.DoubangroupjsTask)
	Store.AddBookTask(doubanbook.DoubanbookTask)
}

// 全局爬虫任务实例
var Store = &CrawlerStore{
	list: []*collect.Task{},
	hash: map[string]*collect.Task{},
}

type CrawlerStore struct {
	list []*collect.Task
	hash map[string]*collect.Task
}

func (c *CrawlerStore) Add(task *collect.Task) {
	c.hash[task.Name] = task
	c.list = append(c.list, task)
}

func (c *CrawlerStore) AddJSTask(m *collect.TaskModel) {
	task := &collect.Task{
		Property: m.Property,
	}

	task.Rule.Root = func() ([]*collect.Request, error) {
		vm := otto.New()
		vm.Set("AddJsReq", AddJsReq)
		v, err := vm.Eval(m.Root)
		if err != nil {
			return nil, err
		}
		e, err := v.Export()
		if err != nil {
			return nil, err
		}
		return e.([]*collect.Request), nil
	}

	for _, r := range m.Rules {
		parseFunc := func(parse string) func(ctx *collect.Context) (collect.ParseResult, error) {
			return func(ctx *collect.Context) (collect.ParseResult, error) {
				vm := otto.New()
				vm.Set("ctx", ctx)
				v, err := vm.Eval(parse)
				if err != nil {
					return collect.ParseResult{}, err
				}
				e, err := v.Export()
				if err != nil {
					return collect.ParseResult{}, err
				}
				return e.(collect.ParseResult), nil
			}
		}(r.ParseFunc)
		if task.Rule.Trunk == nil {
			task.Rule.Trunk = make(map[string]*collect.Rule, 0)
		}
		task.Rule.Trunk[r.Name] = &collect.Rule{
			ParseFunc: parseFunc,
		}
	}

	c.list = append(c.list, task)
	c.hash[task.Name] = task
}

func (c *CrawlerStore) AddBookTask(task *collect.Task) {
	c.list = append(c.list, task)
	c.hash[task.Name] = task
}

// AddJsReq 用于动态规则添加请求
func AddJsReq(jreqs []map[string]interface{}) []*collect.Request {
	reqs := make([]*collect.Request, 0)

	for _, jreq := range jreqs {
		req := &collect.Request{}
		u, ok := jreq["Url"].(string)
		if !ok {
			return nil
		}
		req.Url = u
		req.RuleName, _ = jreq["RuleName"].(string)
		req.Priority, _ = jreq["Priority"].(int)
		req.Method, _ = jreq["Method"].(string)
		reqs = append(reqs, req)
	}
	return reqs
}

type Crawler struct {
	out         chan collect.ParseResult
	Visited     map[string]bool
	VisitedLock sync.Mutex
	failures    map[string]*collect.Request // 失败请求id -> 失败请求
	FailureLock sync.Mutex
	options
}

type Scheduler interface {
	Schedule()
	Push(...*collect.Request)
	Pull() *collect.Request
}

type Schedule struct {
	priReqQueue []*collect.Request
	reqQueue    []*collect.Request
	requestCh   chan *collect.Request
	workerCh    chan *collect.Request
	Logger      *zap.Logger
}

func NewSchedule() *Schedule {
	s := &Schedule{}
	requestCh := make(chan *collect.Request)
	workerCh := make(chan *collect.Request)
	s.requestCh = requestCh
	s.workerCh = workerCh
	return s
}

func (s *Schedule) Schedule() {
	var req *collect.Request
	var ch chan *collect.Request
	for {
		if req == nil && len(s.priReqQueue) > 0 {
			req = s.priReqQueue[0]
			s.priReqQueue = s.priReqQueue[1:]
			ch = s.workerCh
		}
		if req == nil && len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workerCh
		}
		select {
		case r := <-s.requestCh:
			if r.Priority > 0 {
				s.priReqQueue = append(s.priReqQueue, r)
			} else {
				s.reqQueue = append(s.reqQueue, r)
			}
		case ch <- req:
			fmt.Println("dispatch request...")
			req = nil
			ch = nil
		}
	}
}

func (s *Schedule) Push(reqs ...*collect.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}

func (s *Schedule) Pull() *collect.Request {
	r := <-s.workerCh
	return r
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	out := make(chan collect.ParseResult)
	visited := make(map[string]bool)
	failures := make(map[string]*collect.Request)
	e := &Crawler{}
	e.out = out
	e.options = options
	e.Visited = visited
	e.failures = failures
	return e
}

func (e *Crawler) Run() {
	go e.Schedule()
	for i := 0; i < e.WorkCount; i++ {
		go e.CreateWork()
	}
	e.HandleResult()
}

func (e *Crawler) Schedule() {
	var reqs []*collect.Request
	for _, seed := range e.Seeds {
		task := Store.hash[seed.Name]
		task.Fetcher = seed.Fetcher
		//获取初始任务
		rootReqs, _ := task.Rule.Root()
		for _, req := range rootReqs {
			req.Task = task
		}
		reqs = append(reqs, rootReqs...)
	}
	go e.scheduler.Schedule()
	go e.scheduler.Push(reqs...)
}

func (e *Crawler) CreateWork() {
	for {
		r := e.scheduler.Pull()
		if err := r.Check(); err != nil {
			e.Logger.Error("can't fetch", zap.Error(err))
			continue
		}
		if !r.Task.Reload && e.HasVisited(r) {
			e.Logger.Debug("request has visited", zap.String("url:", r.Url))
			continue
		}
		e.StoreVisited(r)
		if r.Task.WaitTime > 0 {
			time.Sleep(r.Task.WaitTime)
		}
		body, err := r.Task.Fetcher.Get(r)
		if len(body) < 6000 {
			e.Logger.Error("can't fetch ", zap.Int("length", len(body)), zap.String("url", r.Url))
			e.SetFailure(r)
			continue
		}
		if err != nil {
			e.Logger.Error("can't fetch", zap.Error(err), zap.String("url", r.Url))
			e.SetFailure(r)
			continue
		}
		//获取当前任务对应的规则
		rule := r.Task.Rule.Trunk[r.RuleName]
		//从规则中获取解析函数解析
		result, err := rule.ParseFunc(&collect.Context{
			Body: body,
			Req:  r,
		})
		if err != nil {
			e.Logger.Error("ParseFunc failed", zap.Error(err), zap.String("url", r.Url))
			e.SetFailure(r)
			continue
		}
		if len(result.Requests) > 0 {
			e.scheduler.Push(result.Requests...)
		}
		e.out <- result
	}
}

func (e *Crawler) HandleResult() {
	for {
		select {
		case result := <-e.out:
			for _, item := range result.Items {
				//todo: store
				e.Logger.Sugar().Info("get result: ", item)
			}
		}
	}
}

func (e *Crawler) HasVisited(r *collect.Request) bool {
	e.VisitedLock.Lock()
	defer e.VisitedLock.Unlock()
	unique := r.Unique()
	return e.Visited[unique]
}

func (e *Crawler) StoreVisited(reqs ...*collect.Request) {
	e.VisitedLock.Lock()
	defer e.VisitedLock.Unlock()
	for _, req := range reqs {
		unique := req.Unique()
		e.Visited[unique] = true
	}
}

func (e *Crawler) SetFailure(req *collect.Request) {
	unique := req.Unique()
	if !req.Task.Reload {
		e.VisitedLock.Lock()
		delete(e.Visited, unique)
		e.VisitedLock.Unlock()
	}
	e.FailureLock.Lock()
	defer e.FailureLock.Unlock()
	if _, ok := e.failures[unique]; !ok {
		// 首次失败时，再重新执行一次
		e.failures[unique] = req
		e.scheduler.Push(req)
	}
	// todo: 失败2次，加载到失败队列中
}
