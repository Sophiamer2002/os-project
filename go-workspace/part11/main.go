package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"os-project/part11/queue"
)

var (
	// Command line arguments
	cap       = flag.Int("cap", 1, "Capacity of the queue")
	n_threads = flag.Int("n-t", 1, "Number of queue operating threads")
	save_dir  = flag.String("dir", "data", "Directory to save the data")

	save_pth  string
	enq_times [][]time.Duration
	deq_times [][]time.Duration

	q      = queue.New[string]()
	record = false // whether to record the data
	run    = true  // whether to continue running
)

func enqThread(i_thread int) {
	enq_times[i_thread] = make([]time.Duration, 0)
	my_enq_times := &enq_times[i_thread]

	// measure the time taken to enqueue
	for run {
		start := time.Now()
		q.Enqueue("")
		if record {
			*my_enq_times = append(*my_enq_times, time.Since(start))
		}
	}
}

func deqThread(i_thread int) {
	deq_times[i_thread] = make([]time.Duration, 0)
	my_deq_times := &deq_times[i_thread]

	// measure the time taken to dequeue
	for run {
		start := time.Now()
		q.Dequeue()
		if record {
			*my_deq_times = append(*my_deq_times, time.Since(start))
		}
	}
}

func main() {
	flag.Parse()
	save_pth = fmt.Sprintf("%s/cap_%d_threads_%d.txt", *save_dir, *cap, *n_threads)
	f, err := os.Create(save_pth)
	if err != nil {
		panic(err)
	}

	q.Init(*cap)
	enq_times = make([][]time.Duration, *n_threads)
	deq_times = make([][]time.Duration, *n_threads)

	for i := 0; i < *n_threads; i++ {
		go enqThread(i)
		go deqThread(i)
	}

	fmt.Printf("Running %d enqueue, %d dequeue threads...\n", *n_threads, *n_threads)

	time.Sleep(10 * time.Microsecond)
	record = true
	time.Sleep(10 * time.Millisecond)
	record = false
	time.Sleep(10 * time.Microsecond)
	run = false

	// save the data
	fmt.Printf("Saving data to %s...\n", save_pth)
	enq_cnt, deq_cnt := 0, 0
	for _, v := range enq_times {
		for _, t := range v {
			enq_cnt += 1
			f.WriteString(fmt.Sprintf("%d ", t.Nanoseconds()))
		}
	}

	f.WriteString("\n")

	for _, v := range deq_times {
		for _, t := range v {
			deq_cnt += 1
			f.WriteString(fmt.Sprintf("%d ", t.Nanoseconds()))
		}
	}

	// write the number of enqueues and dequeues at the beginning of the file
	f.Seek(0, 0)
	f.WriteString(fmt.Sprintf("%d %d\n", enq_cnt, deq_cnt))
	f.Close()
}
