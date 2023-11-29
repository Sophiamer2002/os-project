package main

import (
	"fmt"
	"sync"
	"time"
)

func Println(a ...any) {
	for _, v := range a {
		switch v := v.(type) {
		case int:
			fmt.Println(v, "is an int")
		case string:
			fmt.Println(v, "is a string")
		default:
			fmt.Println(v, "is of unknown type")
		}
	}
}

var (
	wg sync.WaitGroup
)

type test struct {
	s string
	b []byte
}

func testChannel(c chan<- []test) {
	defer wg.Done()
	x := make([]test, 10, 20)
	fmt.Printf("TESTCHANNEL: x address: %p, x capacity: %v, x length: %v\n", &x[0], cap(x), len(x))
	c <- x
	// double x len
	time.Sleep(1 * time.Second)
	u := test{
		s: "test",
		b: []byte("test"),
	}
	x[3].b = u.b
	fmt.Printf("x[3].s address: %p, x[3].b[0] address: %p\n", &x[3].s, &x[3].b[0])
	fmt.Printf("u.s address: %p, u.b address: %p\n", &u.s, &u.b[0])
	fmt.Printf("TESTCHANNEL after: x address: %p, x capacity: %v, x length: %v\n", &x[0], cap(x), len(x))
}

func testDefer(c chan<- int) {
	x := 1
	defer func() {
		c <- x
	}()
	x = 2
}

func main() {
	Println([]any{1, "a", "b"}...)

	// test channel
	c := make(chan []test)
	wg.Add(1)
	go testChannel(c)
	x := <-c
	fmt.Println(x[3], x[4], x[5])
	fmt.Printf("MAIN: x address: %p, x capacity: %v, x length: %v\n", &x[0], cap(x), len(x))
	wg.Wait()
	fmt.Printf("MAIN after: x address: %p, x capacity: %v, x length: %v\n", &x[0], cap(x), len(x))
	fmt.Println(x[3], x[4], x[5])

	// test defer
	c2 := make(chan int)
	go testDefer(c2)
	x2 := <-c2
	fmt.Println(x2)

	if true {
		defer fmt.Println("defer 1")
		fmt.Println("1")
	}
	return
	// fmt.Print("After return\n")
}
