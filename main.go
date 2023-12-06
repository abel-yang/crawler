package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	prime()
}

func pingPong() {
	var Ball int
	table := make(chan int)

	for i := 0; i < 5; i++ {
		go player(i, table)
	}

	table <- Ball

	time.Sleep(1 * time.Second)
	<-table
}

func player(id int, table chan int) {
	for {
		ball := <-table
		fmt.Printf("当前table:%d, 接收值:%d\n", id, ball)
		ball++
		time.Sleep(100 * time.Millisecond)
		table <- ball
	}
}

func search(msg string) chan string {
	ch := make(chan string)
	go func() {
		var i int
		for {
			//模拟找到关键字
			ch <- fmt.Sprintf("get %s %d", msg, i)
			i++
			time.Sleep(1 * time.Second)
		}
	}()
	return ch
}

func fanIn() {
	ch1 := search("abel")
	ch2 := search("jiangshan")

	for {
		select {
		case msg := <-ch1:
			fmt.Println(msg)
		case msg := <-ch2:
			fmt.Println(msg)
		}
	}
}

const (
	WORKERS    = 5
	SUBWORKERS = 3
	TASKS      = 20
	SUBTASKS   = 10
)

func startWorks() {
	var wg sync.WaitGroup
	wg.Add(WORKERS)

	taskCh := make(chan int)

	for i := 0; i < WORKERS; i++ {
		go work(taskCh, &wg)
	}

	for i := 0; i < TASKS; i++ {
		taskCh <- i
	}

	close(taskCh)

	wg.Wait()
}

func work(taskCh chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		task, ok := <-taskCh
		if !ok {
			return
		}

		subtasks := make(chan int)
		for i := 0; i < SUBWORKERS; i++ {
			go subWork(subtasks)
		}

		for i := 0; i < SUBTASKS; i++ {
			task1 := task * i
			subtasks <- task1
		}
	}
}

func subWork(subtasks chan int) {
	task, ok := <-subtasks
	if !ok {
		return
	}
	d := time.Duration(task) * time.Millisecond
	time.Sleep(d)
	fmt.Println("processing task", task)
}

func piped() {
	generator := func(done chan interface{}, integers ...int) chan int {
		intStream := make(chan int)
		go func() {
			defer close(intStream)
			for _, i := range integers {
				select {
				case <-done:
					return
				case intStream <- i:
				}
			}
		}()
		return intStream
	}

	multiply := func(done chan interface{}, intStream chan int, multiplier int) chan int {
		multiplierStream := make(chan int)
		go func() {
			defer close(multiplierStream)
			for i := range intStream {
				select {
				case <-done:
					return
				case multiplierStream <- i * multiplier:
				}
			}
		}()
		return multiplierStream
	}

	add := func(done chan interface{}, intStream chan int, additive int) chan int {
		addStream := make(chan int)
		go func() {
			defer close(addStream)
			for i := range intStream {
				select {
				case <-done:
					return
				case addStream <- i + additive:
				}
			}
		}()
		return addStream
	}

	done := make(chan interface{})
	defer close(done)

	intStream := generator(done, 1, 2, 3, 4)
	multipliedStream := multiply(done, intStream, 2)
	addedStream := add(done, multipliedStream, 1)
	pipeline := multiply(done, addedStream, 2)

	for v := range pipeline {
		fmt.Println(v)
	}

}

// 第一个阶段，数字的生成器
func generator(ch chan int) {
	for i := 2; ; i++ {
		ch <- i
	}
}

// 筛选，排除不能够被prime整除的数
func filter(in chan int, out chan int, prime int) {
	for {
		i := <-in
		if i%prime != 0 {
			out <- i
		}
	}
}

func prime() {
	ch := make(chan int)
	go generator(ch)
	for i := 0; i < 100000; i++ {
		prime := <-ch //获取上一个阶段输出的第一个数，其必然为素数
		fmt.Println(prime)
		ch1 := make(chan int)
		go filter(ch, ch1, prime)
		ch = ch1 // 前一个阶段的输出作为后一个阶段的输入。
	}
}
