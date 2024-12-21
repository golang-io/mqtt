package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

var mu = &sync.Mutex{}
var cond = sync.NewCond(mu)
var done bool

func main() {

	group, _ := errgroup.WithContext(context.Background())

	group.Go(func() error {
		time.Sleep(2 * time.Second)
		mu.Lock()
		done = true
		mu.Unlock()
		cond.Broadcast()
		done = false
		return nil
	})
	group.Go(func() error {

		fmt.Println("b is init")
		mu.Lock()
		fmt.Println("b is locking")
		for !done {
			fmt.Println("b is locked")
			cond.Wait()
		}
		fmt.Println("b is done")
		mu.Unlock()

		return nil
	})
	group.Go(func() error {
		fmt.Println("c is init")
		mu.Lock()
		for !done {
			cond.Wait()
		}
		fmt.Println("c is done")
		mu.Unlock()

		return nil
	})
	if err := group.Wait(); err != nil {
		fmt.Println(err)
	}
	fmt.Println("done")
}
