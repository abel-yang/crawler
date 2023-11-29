package main

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/abel-yang/crawler/collect"
	"github.com/antchfx/htmlquery"
	"regexp"
)

// 正则匹配新闻中的图片
// var headerRe = regexp.MustCompile(`<div class="news_li"[\s\S]*?<h2>[\s\S]*?<a.*?target="_blank">([\s\S]*?</a>)`)
var headerRe = regexp.MustCompile(`<div class="ant-card-body"[\s\S]*?<h2>([\s\S]*?</h2>)`)

func main() {
	url := "https://book.douban.com/subject/1007305/"
	var f collect.Fetcher = collect.BrowserFetch{}
	body, err := f.Get(url)
	if err != nil {
		fmt.Printf("read content failed:%v\n", err)
		return
	}

	fmt.Println(string(body))
}

func css(b []byte) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(b))
	if err != nil {
		fmt.Println("read content failed:%v", err)
	}

	doc.Find("div.small_toplink__GmZhY a[target=_blank]").Each(func(i int, s *goquery.Selection) {
		//获取匹配标签中的文本
		title := s.Text()
		fmt.Printf("review %d: %s\n", i, title)
	})
}

func regMatch(body []byte) {
	matches := headerRe.FindAllSubmatch(body, -1)
	for _, m := range matches {
		fmt.Println("fetch card news: ", string(m[1]))
	}
}

func htmlHandle(b []byte) {
	doc, err := htmlquery.Parse(bytes.NewReader(b))
	if err != nil {
		fmt.Println("htmlquery.Parse failed:%v", err)
	}
	nodes := htmlquery.Find(doc, `//div[@class="small_toplink__GmZhY"]/a[@target="_blank"]/h2`)

	for _, node := range nodes {
		fmt.Println("fetch card ", node.FirstChild.Data)
	}
}
