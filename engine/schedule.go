package engine

import (
	"github.com/abel-yang/crawler/collect"
	"go.uber.org/zap"
	"time"
)

type Schedule struct {
	requestCh chan *collect.Request
	workerCh  chan *collect.Request
	WorkCount int
	Fetcher   collect.Fetcher
	Logger    *zap.Logger
	out       chan collect.ParseResult
	Seeds     []*collect.Request
}

func (s *Schedule) Run() {
	requestCh := make(chan *collect.Request)
	workerCh := make(chan *collect.Request)
	out := make(chan collect.ParseResult)
	s.requestCh = requestCh
	s.workerCh = workerCh
	s.out = out
	go s.Schedule()
	for i := 0; i < s.WorkCount; i++ {
		go s.CreateWork()
	}
	s.HandleResult()
}

func (s *Schedule) Schedule() {
	var reqQueue = s.Seeds
	go func() {
		for {
			var req *collect.Request
			var ch chan *collect.Request
			if len(reqQueue) > 0 {
				req = reqQueue[0]
				reqQueue = reqQueue[1:]
				ch = s.workerCh
			}
			select {
			case r := <-s.requestCh:
				reqQueue = append(reqQueue, r)
			case ch <- req:

			}
		}
	}()
}

func (s *Schedule) CreateWork() {
	for {
		r := <-s.workerCh
		body, err := s.Fetcher.Get(r)
		if err != nil {
			s.Logger.Error("can't fetch", zap.Error(err))
			continue
		}
		result := r.ParseFunc(body)
		s.out <- result
		time.Sleep(r.WaitTime)
	}
}

func (s *Schedule) HandleResult() {
	for {
		select {
		case r := <-s.out:
			for _, req := range r.Requests {
				s.requestCh <- req
			}
			for _, item := range r.Items {
				//todo: store
				s.Logger.Sugar().Info("get result", item)
			}
		}
	}
}