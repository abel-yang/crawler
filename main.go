package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	unicode2 "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io/ioutil"
	"net/http"
	"regexp"
)

// 正则匹配新闻中的图片
// var headerRe = regexp.MustCompile(`<div class="news_li"[\s\S]*?<h2>[\s\S]*?<a.*?target="_blank">([\s\S]*?</a>)`)
var headerRe = regexp.MustCompile(`<div class="ant-card-body"[\s\S]*?<h2>([\s\S]*?</h2>)`)

func main() {
	url := "https://www.thepaper.cn/"
	body, err := fetch(url)

	if err != nil {
		fmt.Printf("read content failed:%v\n", err)
		return
	}

	regMatch(body)
	fmt.Println("--------------")
	htmlHandle(body)
	fmt.Println("css--------------")
	css(body)
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

func fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error status code:%v", resp.StatusCode)
	}
	bodyReader := bufio.NewReader(resp.Body)
	e := DetermineEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return ioutil.ReadAll(utf8Reader)
}

func DetermineEncoding(r *bufio.Reader) encoding.Encoding {
	bytes, err := r.Peek(1024)

	if err != nil {
		fmt.Printf("fetch error:%v", err)
		return unicode2.UTF8
	}

	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}
