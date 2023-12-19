package doubanbook

import (
	"github.com/abel-yang/crawler/collect"
	"regexp"
	"strconv"
	"time"
)

// <a href="/tag/漫画" class="tag">漫画</a>
const tagRe = `<a href="([^"]+)" class="tag">([^<]+)</a>`
const bookListRe = `<a.*?href="([^"]+)" title="([^"]+)"`

var DoubanbookTask = &collect.Task{
	Property: collect.Property{
		Name:     "douban_book_list",
		WaitTime: 1 * time.Second,
		MaxDepth: 5,
		Cookie:   "bid=WOl_3zBUqQg; __yadk_uid=lg51FXJ7EnwHtiwmm7t6tQN0zubL4rpL; _vwo_uuid_v2=DE95D4BD6FD059AEEC660B69F55BEF066|ee5f6b58d49b3b4e96c24d358d16d916; __gads=ID=2b1360ee50e863c4-221629784ae00019:T=1691820585:RT=1691820585:S=ALNI_Mb738jrwgUucjcIvkeX2iZgIREawg; __gpi=UID=00000c7c56f163f4:T=1691820585:RT=1691820585:S=ALNI_MbQj1N1DWKq5M_6yrEZvRKnPRnPaw; viewed=\"1007305_4832380_4272229\"; _pk_id.100001.3ac3=586018a396a19bd6.1687228942.; douban-fav-remind=1; dbcl2=\"174639318:9lcfhTuoiJ0\"; push_noty_num=0; push_doumail_num=0; __utmv=30149280.17463; ct=y; ck=0Orr; __utmc=30149280; __utmz=30149280.1702523773.17.8.utmcsr=time.geekbang.org|utmccn=(referral)|utmcmd=referral|utmcct=/column/article/615675; __utmc=81379588; __utmz=81379588.1702523773.5.4.utmcsr=time.geekbang.org|utmccn=(referral)|utmcmd=referral|utmcct=/column/article/615675; frodotk_db=\"d08301b801229397718905280367bac5\"; _pk_ref.100001.3ac3=%5B%22%22%2C%22%22%2C1702867983%2C%22https%3A%2F%2Ftime.geekbang.org%2Fcolumn%2Farticle%2F615675%3Fcid%3D100124001%22%5D; __utma=30149280.1520868444.1686721274.1702622240.1702867983.20;",
	},
	Rule: collect.RuleTree{
		Root: func() ([]*collect.Request, error) {
			roots := []*collect.Request{
				&collect.Request{
					Priority: 1,
					Url:      "https://book.douban.com",
					Method:   "GET",
					RuleName: "书籍tag",
				},
			}
			return roots, nil
		},
		Trunk: map[string]*collect.Rule{
			"书籍tag": &collect.Rule{ParseFunc: ParseTag},
			"书籍列表":  &collect.Rule{ParseFunc: ParseBookList},
			"书籍简介": &collect.Rule{
				ItemFields: []string{
					"书名", "作者", "页数", "出版社", "得分", "价格", "简介",
				},
				ParseFunc: ParseBookDetail},
		},
	},
}

func ParseTag(ctx *collect.Context) (collect.ParseResult, error) {
	reg := regexp.MustCompile(tagRe)
	matches := reg.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		result.Requests = append(result.Requests, &collect.Request{
			Method:   "GET",
			Task:     ctx.Req.Task,
			Url:      "https://book.douban.com" + string(m[1]),
			Depth:    ctx.Req.Depth + 1,
			RuleName: "书籍列表",
		})
	}
	return result, nil
}

func ParseBookList(ctx *collect.Context) (collect.ParseResult, error) {
	reg := regexp.MustCompile(bookListRe)
	matches := reg.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		req := &collect.Request{
			Method:   "GET",
			Task:     ctx.Req.Task,
			Url:      string(m[1]),
			Depth:    ctx.Req.Depth + 1,
			RuleName: "书籍简介",
		}
		req.TmpData = &collect.Temp{}
		err := req.TmpData.Set("book_name", string(m[2]))
		if err != nil {
			return collect.ParseResult{}, err
		}
		result.Requests = append(result.Requests, req)
	}
	// 在添加limit之前，临时减少抓取数量,防止被服务器封禁
	result.Requests = result.Requests[:3]
	return result, nil
}

var authorRe = regexp.MustCompile(`<span class="pl"> 作者</span>:[\d\D]*?<a.*?>([^<]+)*</a>`)
var publicRe = regexp.MustCompile(`<span class="pl">出版社:</span>([^<]+)<br/>`)
var pageRe = regexp.MustCompile(`<span class="pl">页数:</span> ([^<]+)<br/>`)
var priceRe = regexp.MustCompile(`<span class="pl">定价:</span>([^<]+)<br/>`)
var scoreRe = regexp.MustCompile(`<strong class="ll rating_num " property="v:average">([^<]+)</strong>`)
var intoRe = regexp.MustCompile(`<div class="intro">[\d\D]*?<p>([^<]+)</p></div>`)

func ParseBookDetail(ctx *collect.Context) (collect.ParseResult, error) {
	bookName := ctx.Req.TmpData.Get("book_name")
	page, _ := strconv.Atoi(ExtraString(ctx.Body, pageRe))
	book := map[string]interface{}{
		"书名":  bookName,
		"作者":  ExtraString(ctx.Body, authorRe),
		"页数":  page,
		"出版社": ExtraString(ctx.Body, publicRe),
		"得分":  ExtraString(ctx.Body, scoreRe),
		"价格":  ExtraString(ctx.Body, priceRe),
		"简介":  ExtraString(ctx.Body, intoRe),
	}
	data := ctx.Output(book)
	result := collect.ParseResult{
		Items: []interface{}{data},
	}
	return result, nil
}

func ExtraString(body []byte, re *regexp.Regexp) string {
	match := re.FindSubmatch(body)
	if len(match) >= 2 {
		return string(match[1])
	}
	return ""
}
