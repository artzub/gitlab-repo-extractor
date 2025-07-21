package main

import (
	"context"
	"testing"
	"time"
)

func TestMergeChans(t *testing.T) {
	ctx := context.Background()
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		ch1 <- 1
		ch1 <- 2
		close(ch1)
	}()
	go func() {
		ch2 <- 3
		ch2 <- 4
		close(ch2)
	}()

	out := mergeChans(ctx, ch1, ch2)
	results := make(map[int]struct{})
	for v := range out {
		results[v] = struct{}{}
	}

	for _, want := range []int{1, 2, 3, 4} {
		if _, ok := results[want]; !ok {
			t.Errorf("missing value: %d", want)
		}
	}
}

func TestMergeChans_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan int)

	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			ch <- i
		}
		close(ch)
	}()

	out := mergeChans(ctx, ch)
	cancel()
	read := 0
	for range out {
		read++
	}

	if read > 0 {
		t.Errorf("expected no values to be read, got %d", read)
	}
}

func TestTeeChan_AllChannelsReceiveItems(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := make(chan int)
	amountChans := 3
	chans := teeChan(ctx, in, amountChans)

	amountData := 5

	go func() {
		defer close(in)

		for item := range amountData {
			in <- item
		}
	}()

	for item := range amountData {
		for _, ch := range chans {
			val, ok := <-ch
			if !ok {
				t.Fatalf("Channel closed unexpectedly")
			}
			if val != item {
				t.Errorf("Expected %d, got %d", item, val)
			}
		}
	}

	// After input is closed, all output channels should be closed
	for _, ch := range chans {
		_, ok := <-ch
		if ok {
			t.Errorf("Expected channel to be closed")
		}
	}
}

func TestTeeChan_ContextCancelClosesChannels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan int)
	chans := teeChan(ctx, in, 2)

	cancel() // cancel context immediately

	for _, ch := range chans {
		_, ok := <-ch
		if ok {
			t.Errorf("Expected channel to be closed after context cancel")
		}
	}
}
