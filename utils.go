package main

import (
	"context"
	"sync"
)

func mergeChans[T any](ctx context.Context, chans ...<-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)

		wg := &sync.WaitGroup{}
		wg.Add(len(chans))

		for _, ch := range chans {
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case item, ok := <-ch:
						if !ok {
							return
						}
						select {
						case out <- item:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
		}

		wg.Wait()
	}()

	return out
}

func teeChan[T any](ctx context.Context, in <-chan T, amountChans int) []chan T {
	chans := make([]chan T, amountChans)

	for index := range amountChans {
		chans[index] = make(chan T)
	}

	go func() {
		defer func() {
			for _, ch := range chans {
				close(ch)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-in:
				if !ok {
					return
				}

				wg := &sync.WaitGroup{}

				for _, ch := range chans {
					wg.Add(1)

					go func() {
						defer wg.Done()

						select {
						case ch <- item:
						case <-ctx.Done():
							return
						}
					}()
				}

				wg.Wait()
			}
		}
	}()

	return chans
}
