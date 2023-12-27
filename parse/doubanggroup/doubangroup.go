package doubanggroup

import (
	"fmt"
	"github.com/abel-yang/crawler/spider"
	"regexp"
)

const cityListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`
const ContentRe = `<div class="topic-content">[\s\S]*?房东[\s\S]*?<div`

var DoubangroupTask = &spider.Task{
	Property: spider.Property{
		Name:     "find_douban_sun_room",
		WaitTime: 1,
		MaxDepth: 5,
		Cookie:   "bid=WOl_3zBUqQg; __utmc=30149280; __gads=ID=2b1360ee50e863c4-221629784ae00019:T=1691820585:RT=1691820585:S=ALNI_Mb738jrwgUucjcIvkeX2iZgIREawg; __gpi=UID=00000c7c56f163f4:T=1691820585:RT=1691820585:S=ALNI_MbQj1N1DWKq5M_6yrEZvRKnPRnPaw; viewed=\"1007305_4832380_4272229\"; _pk_id.100001.8cb4=3c64d811c51e5049.1701920508.; __yadk_uid=mGSGRfXOuU26e20uLUE0uJxbPm5qGM7z; douban-fav-remind=1; dbcl2=\"174639318:9lcfhTuoiJ0\"; ck=0Orr; push_noty_num=0; push_doumail_num=0; __utmz=30149280.1702275071.10.6.utmcsr=open.weixin.qq.com|utmccn=(referral)|utmcmd=referral|utmcct=/; __utmv=30149280.17463; _pk_ref.100001.8cb4=%5B%22%22%2C%22%22%2C1702367140%2C%22https%3A%2F%2Fopen.weixin.qq.com%2F%22%5D; _pk_ses.100001.8cb4=1; __utma=30149280.1520868444.1686721274.1702351542.1702367140.12; __utmt=1; __utmb=30149280.11.5.1702367410037",
	},
	Rule: spider.RuleTree{
		Root: func() ([]*spider.Request, error) {
			var roots []*spider.Request
			for i := 0; i <= 100; i += 25 {
				str := fmt.Sprintf("https://www.douban.com/group/szsh/discussion?start=%d", i)
				roots = append(roots, &spider.Request{
					Url:      str,
					Method:   "GET",
					Priority: 1,
					RuleName: "解析网站URL",
				})
			}

			return roots, nil
		},
		Trunk: map[string]*spider.Rule{
			"解析网站URL": &spider.Rule{ParseFunc: ParseURL},
			"解析阳台房":   &spider.Rule{ParseFunc: GetSunRoom},
		},
	},
}

func GetSunRoom(ctx *spider.Context) (spider.ParseResult, error) {
	re := regexp.MustCompile(ContentRe)
	if ok := re.Match(ctx.Body); !ok {
		return spider.ParseResult{
			Items: []interface{}{},
		}, nil
	}

	result := spider.ParseResult{
		Items: []interface{}{ctx.Req.Url},
	}
	return result, nil
}

func ParseURL(ctx *spider.Context) (spider.ParseResult, error) {
	re := regexp.MustCompile(cityListRe)

	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := spider.ParseResult{}
	for _, m := range matches {
		url := string(m[1])
		result.Requests = append(result.Requests, &spider.Request{
			Method:   "GET",
			Url:      url,
			Task:     ctx.Req.Task,
			Depth:    ctx.Req.Depth + 1,
			RuleName: "解析阳台房",
		})
	}
	return result, nil
}
