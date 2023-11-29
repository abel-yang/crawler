package main

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

type Student struct {
	Age  int
	Name string
}

type Trade struct {
	tradeId int
	Price   int
}

func main() {
	ctx()
	//createQuery(Student{Age: 20, Name: "abel"})
	//createQuery(Trade{tradeId: 100, Price: 20})
}

func ctx() {
	//父context退出 会导致所有子context退出，子context退出不会影响父context
	ctx := context.Background()
	before := time.Now()
	preCtx, _ := context.WithTimeout(ctx, 500*time.Millisecond)
	go func() {
		childCtx, _ := context.WithTimeout(preCtx, 300*time.Millisecond)
		select {
		case <-childCtx.Done():
			after := time.Now()
			fmt.Println("child during:", after.Sub(before).Milliseconds())
		}
	}()
	select {
	case <-preCtx.Done():
		after := time.Now()
		fmt.Println("child during:", after.Sub(before).Milliseconds())
	}
}

func createQuery(q interface{}) string {
	//判断类型为结构体
	if reflect.ValueOf(q).Kind() == reflect.Struct {
		//获取结构体名字
		t := reflect.TypeOf(q).Name()
		//查询语句
		query := fmt.Sprintf("insert into %s values(", t)
		v := reflect.ValueOf(q)
		//遍历结构体
		for i := 0; i < v.NumField(); i++ {
			//判断结构体类型
			switch v.Field(i).Kind() {
			case reflect.Int:
				if i == 0 {
					query = fmt.Sprintf("%s%d", query, v.Field(i).Int())
				} else {
					query = fmt.Sprintf("%s, %d", query, v.Field(i).Int())
				}
			case reflect.String:
				if i == 0 {
					query = fmt.Sprintf("%s\"%s\"", query, v.Field(i).String())
				} else {
					query = fmt.Sprintf("%s, \"%s\"", query, v.Field(i).String())
				}
			}
		}
		query = fmt.Sprintf("%s)", query)
		fmt.Println(query)
		return query
	}
	return ""
}
