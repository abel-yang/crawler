package doubanggroup

import (
	"github.com/abel-yang/crawler/collect"
	"regexp"
)

const cityListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`

func ParseUrl(contents []byte) collect.ParseResult {
	re := regexp.MustCompile(cityListRe)

	matches := re.FindAllSubmatch(contents, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(
			result.Requests, &collect.Request{
				Url: u,
				ParseFunc: func(c []byte) collect.ParseResult {
					return GetContent(c, u)
				},
			},
		)
	}
	return result
}

const ContentRe = `<div class="topic-content">[\s\S]*?房东[\s\S]*?<div`

func GetContent(contents []byte, url string) collect.ParseResult {
	re := regexp.MustCompile(ContentRe)

	ok := re.Match(contents)
	if !ok {
		return collect.ParseResult{
			Items: []interface{}{},
		}
	}

	result := collect.ParseResult{
		Items: []interface{}{url},
	}
	return result
}
