package main

import (
	"sync"
	"testing"
)

func TestProgressCounter(t *testing.T) {
	total := 10
	counter := NewProgressCounter(uint32(total))
	wg := &sync.WaitGroup{}
	wg.Add(total)

	for i := 0; i < total; i++ {
		go func(i int) {
			defer wg.Done()

			success := i%2 == 0
			counter.Update(success)
		}(i)
	}

	wg.Wait()

	totalRes, completed, success, failed := counter.GetStats()

	halfTotal := uint32(total / 2)

	if totalRes != uint32(total) ||
		completed != uint32(total) ||
		success != halfTotal ||
		failed != halfTotal {
		t.Fatalf("Test failed: stats do not match expected values")
	}

	errors := counter.GetErrors()
	if errors != halfTotal {
		t.Fatalf("Test failed: expected %d errors, got %d", halfTotal, errors)
	}
}
