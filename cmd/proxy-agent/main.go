package main

import (
	"context"

	"github.com/ojbkgo/aproxy/pkg/proxy"
)

func init() {

}

func main() {
	cm := &proxy.ClientManager{
		Port:       81,
		RemoteAddr: "127.0.0.1",
		RemotePort: 10000,
	}

	err := cm.Dial(context.Background())
	if err != nil {
		panic(err)
	}

	err = cm.WaitConnection(context.Background())
	if err != nil {
		panic(err)
	}
}
