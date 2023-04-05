package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/ojbkgo/aproxy/pkg/proxy"
)

func init() {

}

func main() {
	cm := proxy.NewClient()

	cm.Info = &proxy.AppInfo{
		Name: "test-local",
	}
	ch := make(chan os.Signal)
	signal.Notify(ch)

	go func() {
		select {
		case <-ch:
			cm.Quit()
			os.Exit(0)
		}
	}()

	err := cm.Register(context.Background(), "127.0.0.1:10000,10001", 8201)
	if err != nil {
		panic(err)
	}

	err = cm.ProxyTo(context.Background(), "127.0.0.1:82")
	if err != nil {
		panic(err)
	}
}
