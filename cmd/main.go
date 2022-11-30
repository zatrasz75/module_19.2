package main

import (
	"fmt"
	"sync"
)

// Число сообщений от источника
const messagesAmountPerGoroutine int = 5

// Функция разуплотнения каналов
func demultiplexingFunc(dataSourceChan chan int, amount int) ([]chan int, <-chan int) {
	var output = make([]chan int, amount)
	var done = make(chan int)

	for i := range output {
		output[i] = make(chan int)
	}
	go func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			for v := range dataSourceChan {
				for _, c := range output {
					c <- v
				}
			}
		}()
		wg.Wait()
		close(done)
	}()
	return output, done
}

// Функция уплотнения каналов
func multiplexingFunc(done <-chan int, channels ...chan int) <-chan int {
	var wg sync.WaitGroup

	multiplexedChan := make(chan int)
	multiplex := func(c <-chan int) {
		defer wg.Done()
		for {
			select {
			case i := <-c:
				multiplexedChan <- i

			case <-done:
				return
			}
		}
	}
	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}
	// Запускаем горутину, которая закроет канал после того,
	// как в закрывающий канал поступит сигнал о прекращении
	// работы всех
	go func() {
		wg.Wait()
		close(multiplexedChan)
	}()
	return multiplexedChan
}

func main() {
	// Горутина - источник данных
	// Функция создает свой собственный канал
	// и посылает в него пять сообщений
	startDataSource := func() chan int {
		c := make(chan int)
		go func() {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 1; i <= messagesAmountPerGoroutine; i++ {
					c <- i
				}
			}()
			wg.Wait()
			close(c)
		}()
		return c
	}
	// Запускаем источник данных и уплотняем каналы
	consumers, done := demultiplexingFunc(startDataSource(), 5)

	c := multiplexingFunc(done, consumers...)
	// Централизованно получаем сообщения от всех нужных нам
	// источников данных
	for data := range c {
		fmt.Println(data)
	}
}
