package utils

import (
	"fmt"
	"io"
	"net"
	"sync"
)

func ForwardData(conn1 net.Conn, conn2 net.Conn) chan struct{} {
	closed := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer func() {
			conn1.Close()
			conn2.Close()
			wg.Done()
			fmt.Println("finish 2")
		}()

		if conn1 == nil || conn2 == nil {
			return
		}

		io.Copy(conn2, conn1)
	}()

	go func() {
		defer func() {
			conn1.Close()
			conn2.Close()
			wg.Done()
			fmt.Println("finish 1")
		}()

		if conn1 == nil || conn2 == nil {
			return
		}

		io.Copy(conn1, conn2)
	}()

	go func() {
		wg.Wait()
		close(closed)
	}()

	return closed
}
