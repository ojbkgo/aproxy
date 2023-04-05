/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ojbkgo/aproxy/pkg/proxy"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "proxy client",
	Long:  `proxy client`,
	Run: func(cmd *cobra.Command, args []string) {
		ps := strings.Split(*cArg.proxy, ":")
		local := strings.Join(ps[1:], ":")
		expose, _ := strconv.ParseInt(ps[0], 10, 64)

		client := proxy.NewClient()

		ch := make(chan os.Signal)
		signal.Notify(ch)

		go func() {
			select {
			case <-ch:
				client.Quit()
				os.Exit(0)
			}
		}()

		err := client.Register(context.Background(), *cArg.addr, uint(expose))
		if err != nil {
			fmt.Println(err)
			return
		}

		err = client.ProxyTo(context.Background(), local)
		if err != nil {
			fmt.Println(err)
			return
		}
	},
}

type clientArgs struct {
	addr  *string
	proxy *string
}

var cArg clientArgs

func init() {
	rootCmd.AddCommand(clientCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clientCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	cArg.addr = clientCmd.Flags().String("addr", "127.0.0.1:80", "proxy server address")
	cArg.proxy = clientCmd.Flags().String("proxy", "8200:127.0.0.1:80", "proxy to local address: {expose port}:{local ip}:{local port}")
}
