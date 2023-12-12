package engine

import (
	"fmt"
	"github.com/abel-yang/crawler/collect"
	"go.uber.org/zap"
)

type Crawler struct {
	out chan collect.ParseResult
	options
}

type Scheduler interface {
	Schedule()
	Push(...*collect.Request)
	Pull() *collect.Request
}

type Schedule struct {
	reqQueue  []*collect.Request
	requestCh chan *collect.Request
	workerCh  chan *collect.Request
	Logger    *zap.Logger
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
	for {
		var req *collect.Request
		var ch chan *collect.Request
		if len(s.reqQueue) > 0 {
			req = s.reqQueue[0]
			s.reqQueue = s.reqQueue[1:]
			ch = s.workerCh
		}
		select {
		case r := <-s.requestCh:
			s.reqQueue = append(s.reqQueue, r)
		case ch <- req:
			fmt.Println("dispatch request...")
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
	e := &Crawler{}
	e.out = out
	e.options = options
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
		seed.RootReq.Url = seed.Url
		seed.RootReq.Task = seed
		reqs = append(reqs, seed.RootReq)
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
		body, err := r.Task.Fetcher.Get(r)
		if len(body) < 6000 {
			e.Logger.Error("can't fetch ", zap.Int("length", len(body)), zap.String("url", r.Url))
			continue
		}
		if err != nil {
			e.Logger.Error("can't fetch", zap.Error(err), zap.String("url", r.Url))
			continue
		}
		result := r.ParseFunc(body, r)
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
