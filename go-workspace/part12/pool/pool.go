package pool

// 参考代码：https://github.com/wazsmwazsm/mortar

import (
	"sync"

	"os-project/part11/queue"
)

type Task struct {
	Handler func(...interface{})
	Params  []interface{}
}

type Pool struct {
	n_worker int
	wg       sync.WaitGroup
	q        *queue.Queue[*Task]
}

func New(n, q_cap int) *Pool {
	q := queue.New[*Task]()
	q.Init(q_cap)
	return &Pool{
		n_worker: n,
		q:        q,
	}
}

func (p *Pool) Run() {
	p.wg.Add(p.n_worker)
	for i := 0; i < p.n_worker; i++ {
		go func() {
			defer p.wg.Done()
			for {
				t, ok := p.q.Dequeue()
				if ok == 0 {
					break
				}
				if t == nil || ok != 1 {
					panic("The task shouldn't be none!")
				}
				t.Handler(t.Params...)
			}
		}()
	}
}

func (p *Pool) AddTask(t *Task) {
	p.q.Enqueue(t)
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) Close() {
	p.q.Close()
}
