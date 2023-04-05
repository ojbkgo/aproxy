/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/ojbkgo/aproxy/pkg/proxy"
	"github.com/ojbkgo/aproxy/pkg/utils"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		ip, ports := utils.ParseAddr(*serverArg.Bind)
		m := proxy.NewServer()
		go func() {
			log.Print("start control channel")
			err := m.Run(context.Background(), fmt.Sprintf("%s:%d", ip, ports[0]))
			if err != nil {
				panic(err)
			}
		}()

		go func() {
			log.Println("start data channel")
			err := m.RunDataChannel(context.TODO(), fmt.Sprintf("%s:%d", ip, ports[1]))
			if err != nil {
				panic(err)
			}
		}()

		m.Stats()
		m.Wait()
	},
}

type ServerArgs struct {
	Bind *string
}

var serverArg = ServerArgs{}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverArg.Bind = serverCmd.Flags().String("bind", "0.0.0.0:10000,10001", "bind address of channel, default 0.0.0.0:10000,10001")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
