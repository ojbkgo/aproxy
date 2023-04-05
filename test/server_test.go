package test

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func Test_Server(t *testing.T) {
	lsn, _ := net.Listen("tcp", ":12345")
	for {
		conn, _ := lsn.Accept()
		go fn(conn)
		go close(conn, time.Second*10)
	}
}

func fn(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err == io.EOF {
			fmt.Println("eof")
			break
		}
		fmt.Println("size: ", n, "error:", err, "data:", string(buf[0:n]))
	}
}

func close(conn net.Conn, duration time.Duration) {
	time.Sleep(duration)
	conn.Close()
}

func Test_Client(t *testing.T) {
	conn, _ := net.Dial("tcp", ":12345")

	fn(conn)
}
