package collect

import (
	"bufio"
	"context"
	"fmt"
	"github.com/abel-yang/crawler/proxy"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	unicode2 "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Fetcher interface {
	Get(url string) ([]byte, error)
}

type BaseFetch struct {
}

type BrowserFetch struct {
	Timeout time.Duration
	Proxy   proxy.ProxyFunc
}

// Get 模拟浏览器访问
func (b BrowserFetch) Get(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: b.Timeout,
	}

	if b.Proxy != nil {
		transport := http.DefaultTransport.(*http.Transport)
		transport.Proxy = b.Proxy
		client.Transport = transport
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("get url failed:%v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	bodyReader := bufio.NewReader(resp.Body)
	e := DetermineEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return ioutil.ReadAll(utf8Reader)
}

/*
*
首先我们导入了 chromedp 库，并调用 chromedp.NewContext 为我们创建了一个浏览器的实例。
它的实现原理非常简单，即查找当前系统指定路径下指定的谷歌应用程序，
并默认用无头模式（Headless 模式）启动谷歌浏览器实例。
通过无头模式，我们肉眼不会看到谷歌浏览器窗口的打开过程，但它确实已经在后台运行了。
*/
func chromeFetch() {
	//1、创建谷歌浏览器实例
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	//2.设置context超时时间
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)

	defer cancel()

	//3.爬取页面，等待某一个元素出现，接着模拟鼠标点击，最后获取数据
	//chromedp.WaitVisible 指的是“等待当前标签可见”，其参数使用的是 CSS 选择器的形式。在这个例子中，body > footer 标签可见，代表正文已经加载完毕
	//chromedp.Click 指的是“模拟对某一个标签的点击事件”。
	//chromedp.Value 用于获取指定标签的数据。
	var example string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://pkg.go.dev/time`),
		chromedp.WaitVisible(`body > footer`),
		chromedp.Click(`#example-After`, chromedp.NodeVisible),
		chromedp.Value(`#example-After textarea`, &example),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Go's time.after example:\n%s", example)
}

func (BaseFetch) Get(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error status code:%d", resp.StatusCode)
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
