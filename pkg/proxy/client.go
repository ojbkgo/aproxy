package proxy

import (
	"context"
	"net"
	"sync"
)

func NewClientManager(port uint) *ClientManager {
	m := &ClientManager{
		Port:  port,
		peers: make(map[uint64]*clientPeer),
	}

	return m
}

type ClientManager struct {
	Port    uint
	proxyID uint64
	ctl     net.Conn
	peers   map[uint64]*clientPeer
	mu      sync.Mutex
}

func (c *ClientManager) Dial(ctx context.Context, addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	msg := createMessage(MessageTypeRegister, &MessageRegister{
		Port: c.Port,
	})

	_, err = msg.Write(conn)
	if err != nil {
		return err
	}

	rawMsg, err := readMessage(conn)
	if err != nil {
		return err
	}

	var ack *MessageRegisterAck
	if v, ok := assertMessage(rawMsg).(*MessageRegisterAck); !ok {
		return ErrBadConnection
	} else {
		ack = v
	}

	c.proxyID = ack.ID
	c.ctl = conn

	return nil
}

func (c *ClientManager) DialTLS(ctx context.Context, addr string) error {
	return nil
}

func (c *ClientManager) register() {

}

func (c *ClientManager) waitConnection(ctx context.Context, conn net.Conn) error {
	return nil
}

type clientPeer struct {
	connID uint64
	a      net.Conn
	b      net.Conn
}

func (p *clientPeer) startTransfer() {

}
