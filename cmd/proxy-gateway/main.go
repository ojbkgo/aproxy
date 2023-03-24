package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ojbkgo/aproxy/pkg/proxy"
)

func main() {
	m := proxy.NewManager()
	go func() {
		fmt.Println("start ctl...")
		err := m.Run(context.Background(), ":10000")
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		fmt.Println("start data...")
		err := m.RunDataChannel(context.TODO(), ":10001")
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Minute * 10)
}
