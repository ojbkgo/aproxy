package main

import (
	"context"

	"github.com/ojbkgo/aproxy/pkg/proxy"
)

func init() {

}

func main() {

	cm := proxy.NewClientManager(81)
	cm.RemoteAddr = "127.0.0.1"
	cm.RemotePort = 10000

	err := cm.Dial(context.Background())
	if err != nil {
		panic(err)
	}

	err = cm.WaitConnection(context.Background())
	if err != nil {
		panic(err)
	}
}
