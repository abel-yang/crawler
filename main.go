package main

import (
	"github.com/abel-yang/crawler/log"
	"go.uber.org/zap/zapcore"
	"regexp"
)

// 正则匹配新闻中的图片
// var headerRe = regexp.MustCompile(`<div class="news_li"[\s\S]*?<h2>[\s\S]*?<a.*?target="_blank">([\s\S]*?</a>)`)
var headerRe = regexp.MustCompile(`<div class="ant-card-body"[\s\S]*?<h2>([\s\S]*?</h2>)`)

func main() {
	plugin, c := log.NewFilePlugin("./log.txt", zapcore.InfoLevel)
	defer c.Close()
	logger := log.NewLogger(plugin)
	logger.Info("log init end")
}
