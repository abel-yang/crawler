package collect

type Rule struct {
	ParseFunc func(*Context) (ParseResult, error) // 内容解析函数
}

type RuleTree struct {
	Root  func() ([]*Request, error) //根节点（执行入口）
	Trunk map[string]*Rule
}
