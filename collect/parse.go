package collect

type Rule struct {
	ParseFunc func(*Context) ParseResult // 内容解析函数
}

type RuleTree struct {
	Root  func() []*Request //根节点（执行入口）
	Trunk map[string]*Rule
}

type Context struct {
	Body []byte
	Req  *Request
}
