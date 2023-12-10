package main

import (
	"fmt"
	"sort"
)

// 计算机课程和其前序课程的映射关系
var prereqs = map[string][]string{
	"algorithms": {"data structures"},
	"calculus":   {"linear algebra"},

	"compilers": {
		"data structures",
		"formal languages",
		"computer organization",
	},

	"data structures":       {"discrete math"},
	"databases":             {"data structures"},
	"discrete math":         {"intro to programming"},
	"formal languages":      {"discrete math"},
	"networks":              {"operating systems"},
	"operating systems":     {"data structures", "computer organization"},
	"programming languages": {"data structures", "computer organization"},
}

func main() {
	for i, course := range topSort(prereqs) {
		fmt.Printf("%d:\t%s\n", i+1, course)
	}
}

// 深度优先遍历
func topSort(m map[string][]string) []string {
	var order []string
	seen := make(map[string]bool)
	var visitAll func(items []string)

	visitAll = func(items []string) {
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				visitAll(m[item])
				order = append(order, item)
			}
		}
	}

	var keys []string
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	visitAll(keys)
	return order
}

func breadthFirst(urls []string) {
	for len(urls) > 0 {
		items := urls
		urls = nil
		for _, item := range items {
			urls = append(urls, exactUrl(item))
		}
	}
}

func exactUrl(item string) string {

	return ""
}
