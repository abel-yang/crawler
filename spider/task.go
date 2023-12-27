package spider

import "sync"

type Task struct {
	Visited     map[string]bool
	VisitedLock sync.Mutex
	Rule        RuleTree
	Options
}

type Property struct {
	Name     string `json:"name"` // 任务名称，应保证唯一性
	Url      string `json:"url"`
	Cookie   string `json:"cookie"`
	WaitTime int64  `json:"waitTime"` // 随机休眠时间，秒
	Reload   bool   `json:"reload"`   //网页是否可以重复爬取
	MaxDepth int    `json:"max_depth"`
}

type TaskConfig struct {
	Name     string
	Cookie   string
	WaitTime int64
	Reload   bool
	MaxDepth int64
	Fetcher  string
	Limits   []LimitConfig
}

type LimitConfig struct {
	EventCount int
	EventDur   int //秒
	Bucket     int //桶大小
}

type Fetcher interface {
	Get(request *Request) ([]byte, error)
}

func NewTask(opts ...Option) *Task {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	t := &Task{}
	t.Options = options
	return t
}
