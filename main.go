package main

import (
	"fmt"
	"github.com/abel-yang/crawler/collect"
	"github.com/abel-yang/crawler/engine"
	"github.com/abel-yang/crawler/log"
	"github.com/abel-yang/crawler/parse/doubanggroup"
	"github.com/abel-yang/crawler/proxy"
	"go.uber.org/zap/zapcore"
	"time"
)

func main() {
	plugin := log.NewStdoutPlugin(zapcore.InfoLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	proxyUrl := []string{"http://127.0.0.1:9981", "http://127.0.0.1:9981"}
	p, err := proxy.RoundRobinProxySwitcher(proxyUrl)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed")
		return
	}
	var f collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}
	cookie := "bid=WOl_3zBUqQg; __utmc=30149280; __gads=ID=2b1360ee50e863c4-221629784ae00019:T=1691820585:RT=1691820585:S=ALNI_Mb738jrwgUucjcIvkeX2iZgIREawg; __gpi=UID=00000c7c56f163f4:T=1691820585:RT=1691820585:S=ALNI_MbQj1N1DWKq5M_6yrEZvRKnPRnPaw; viewed=\"1007305_4832380_4272229\"; _pk_id.100001.8cb4=3c64d811c51e5049.1701920508.; __yadk_uid=mGSGRfXOuU26e20uLUE0uJxbPm5qGM7z; douban-fav-remind=1; ap_v=0,6.0; dbcl2=\"174639318:9lcfhTuoiJ0\"; ck=0Orr; _pk_ref.100001.8cb4=%5B%22%22%2C%22%22%2C1702275071%2C%22https%3A%2F%2Fopen.weixin.qq.com%2F%22%5D; _pk_ses.100001.8cb4=1; push_noty_num=0; push_doumail_num=0; __utma=30149280.1520868444.1686721274.1702270955.1702275071.10; __utmz=30149280.1702275071.10.6.utmcsr=open.weixin.qq.com|utmccn=(referral)|utmcmd=referral|utmcct=/; __utmt=1; __utmv=30149280.17463; __utmb=30149280.7.5.1702275071"

	var seeds = make([]*collect.Task, 0, 1000)
	for i := 0; i <= 100; i += 25 {
		str := fmt.Sprintf("https://www.douban.com/group/szsh/discussion?start=%d", i)
		seeds = append(seeds, &collect.Task{
			Url:      str,
			Cookie:   cookie,
			WaitTime: 1 * time.Second,
			Fetcher:  f,
			MaxDepth: 1024,
			RootReq: &collect.Request{
				ParseFunc: doubanggroup.ParseUrl,
			},
		})
	}

	s := engine.NewSchedule(
		engine.WithFetcher(f),
		engine.WithWorkCount(5),
		engine.WithLogger(logger),
		engine.WithSeeds(seeds),
	)

	s.Run()
}
