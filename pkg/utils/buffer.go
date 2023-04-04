package utils

import (
	"fmt"
	"io"
	"net"
)

func ForwardData(conn1 net.Conn, conn2 net.Conn) {
	go func() {
		defer conn1.Close()
		defer conn2.Close()
		size, err := io.Copy(conn2, conn1)
		fmt.Println(size, err)
	}()

	go func() {
		defer conn1.Close()
		defer conn2.Close()
		size, err := io.Copy(conn1, conn2)
		fmt.Println(size, err)
	}()
}
