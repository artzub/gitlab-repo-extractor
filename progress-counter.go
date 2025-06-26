package main

import "sync"

type ProgressCounter struct {
	total     uint32
	completed uint32
	success   uint32
	failed    uint32
	mutex     *sync.Mutex
}

func NewProgressCounter(total uint32) *ProgressCounter {
	counter := &ProgressCounter{
		total: total,
		mutex: &sync.Mutex{},
	}

	return counter
}

func (c *ProgressCounter) Update(success bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.completed++
	if success {
		c.success++
	} else {
		c.failed++
	}
}

func (c *ProgressCounter) GetStats() (uint32, uint32, uint32, uint32) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.total,
		c.completed,
		c.success,
		c.failed
}
