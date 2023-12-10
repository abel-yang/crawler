package collect

type ParseResult struct {
	Requests []*Request    // 网站获取到的资源链接
	Items    []interface{} //网站获取到的数据
}

type Request struct {
	Url       string
	Cookie    string
	ParseFunc func([]byte) ParseResult
}
