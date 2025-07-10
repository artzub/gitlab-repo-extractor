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
