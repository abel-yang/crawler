package engine

import (
	"fmt"
	"github.com/abel-yang/crawler/parse/doubanbook"
	"github.com/abel-yang/crawler/parse/doubanggroup"
	"github.com/abel-yang/crawler/spider"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"runtime/debug"
	"sync"
)

func init() {
	Store.Add(doubanggroup.DoubangroupTask)
	Store.AddJSTask(doubanggroup.DoubangroupjsTask)
	Store.AddBookTask(doubanbook.DoubanbookTask)
}

// 全局爬虫任务实例
var Store = &CrawlerStore{
	list: []*spider.Task{},
	Hash: map[string]*spider.Task{},
}

func GetFields(taskName string, ruleName string) []string {
	return Store.Hash[taskName].Rule.Trunk[ruleName].ItemFields
}

type CrawlerStore struct {
	list []*spider.Task
	Hash map[string]*spider.Task
}

func (c *CrawlerStore) Add(task *spider.Task) {
	c.Hash[task.Name] = task
	c.list = append(c.list, task)
}

func (c *CrawlerStore) AddJSTask(m *spider.TaskModel) {
	task := &spider.Task{
		Property: m.Property,
	}

	task.Rule.Root = func() ([]*spider.Request, error) {
		vm := otto.New()
		err := vm.Set("AddJsReq", AddJsReq)
		if err != nil {
			return nil, err
		}
		v, err := vm.Eval(m.Root)
		if err != nil {
			return nil, err
		}
		e, err := v.Export()
		if err != nil {
			return nil, err
		}
		return e.([]*spider.Request), nil
	}

	for _, r := range m.Rules {
		parseFunc := func(parse string) func(ctx *spider.Context) (spider.ParseResult, error) {
			return func(ctx *spider.Context) (spider.ParseResult, error) {
				vm := otto.New()
				err := vm.Set("ctx", ctx)
				if err != nil {
					return spider.ParseResult{}, err
				}
				v, err := vm.Eval(parse)
				if err != nil {
					return spider.ParseResult{}, err
				}
				e, err := v.Export()
				if err != nil {
					return spider.ParseResult{}, err
				}
				return e.(spider.ParseResult), nil
			}
		}(r.ParseFunc)
		if task.Rule.Trunk == nil {
			task.Rule.Trunk = make(map[string]*spider.Rule, 0)
		}
		task.Rule.Trunk[r.Name] = &spider.Rule{
			ParseFunc: parseFunc,
		}
	}

	c.list = append(c.list, task)
	c.Hash[task.Name] = task
}

func (c *CrawlerStore) AddBookTask(task *spider.Task) {
	c.list = append(c.list, task)
	c.Hash[task.Name] = task
}

// AddJsReq 用于动态规则添加请求
func AddJsReq(jreqs []map[string]interface{}) []*spider.Request {
	reqs := make([]*spider.Request, 0)

	for _, jreq := range jreqs {
		req := &spider.Request{}
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
	out         chan spider.ParseResult
	Visited     map[string]bool
	VisitedLock sync.Mutex
	failures    map[string]*spider.Request // 失败请求id -> 失败请求
	FailureLock sync.Mutex
	options
}

type Scheduler interface {
	Schedule()
	Push(reqs ...*spider.Request)
	Pull() *spider.Request
}

type Schedule struct {
	priReqQueue []*spider.Request
	reqQueue    []*spider.Request
	requestCh   chan *spider.Request
	workerCh    chan *spider.Request
	Logger      *zap.Logger
}

func NewSchedule() *Schedule {
	s := &Schedule{}
	requestCh := make(chan *spider.Request)
	workerCh := make(chan *spider.Request)
	s.requestCh = requestCh
	s.workerCh = workerCh
	return s
}

func (s *Schedule) Schedule() {
	var req *spider.Request
	var ch chan *spider.Request
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

func (s *Schedule) Push(reqs ...*spider.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}

func (s *Schedule) Pull() *spider.Request {
	r := <-s.workerCh

	return r
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	out := make(chan spider.ParseResult)
	visited := make(map[string]bool)
	failures := make(map[string]*spider.Request)
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
	var reqs []*spider.Request
	for _, seed := range e.Seeds {
		task := Store.Hash[seed.Name]
		task.Fetcher = seed.Fetcher
		task.Storage = seed.Storage
		task.Limit = seed.Limit
		task.Logger = e.Logger
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
	defer func() {
		if err := recover(); err != nil {
			e.Logger.Error("worker panic", zap.Any("err", err), zap.String("stack", string(debug.Stack())))
		}
	}()
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

		body, err := r.Fetch()
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
		result, err := rule.ParseFunc(&spider.Context{
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
	for result := range e.out {
		for _, item := range result.Items {
			switch d := item.(type) {
			case *spider.DataCell:
				name := d.GetTaskName()
				task := Store.Hash[name]

				if err := task.Storage.Save(d); err != nil {
					e.Logger.Error("存储数据出错", zap.Error(err))
				}
			}
			e.Logger.Sugar().Info("get result: ", item)
		}
	}
}

func (e *Crawler) HasVisited(r *spider.Request) bool {
	e.VisitedLock.Lock()
	defer e.VisitedLock.Unlock()
	unique := r.Unique()
	return e.Visited[unique]
}

func (e *Crawler) StoreVisited(reqs ...*spider.Request) {
	e.VisitedLock.Lock()
	defer e.VisitedLock.Unlock()
	for _, req := range reqs {
		unique := req.Unique()
		e.Visited[unique] = true
	}
}

func (e *Crawler) SetFailure(req *spider.Request) {
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
