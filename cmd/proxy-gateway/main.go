package main

import (
	"context"
	"log"
	"time"

	"github.com/ojbkgo/aproxy/pkg/proxy"
)

func main() {
	m := proxy.NewServer()
	go func() {
		log.Print("start control channel")
		err := m.Run(context.Background(), ":10000")
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		log.Println("start data channel")
		err := m.RunDataChannel(context.TODO(), ":10001")
		if err != nil {
			panic(err)
		}
	}()

	m.Stats()

	time.Sleep(time.Minute * 10)
}
