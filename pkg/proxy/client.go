package proxy

import (
	"context"
	"net"
	"sync"
)

type ClientManager struct {
	Port    uint
	proxyID uint64
	ctl     net.Conn
	peers   map[uint64]*clientPeer
	mu      sync.Mutex
}

func (c *ClientManager) Dial(ctx context.Context, addr string) error {

}

func (c *ClientManager) DialTLS(ctx context.Context, addr string) error {

}

func (c *ClientManager) register() {

}

func (c *ClientManager) waitConnection(ctx context.Context, conn net.Conn) error {

}

type clientPeer struct {
	connID uint64
	a      net.Conn
	b      net.Conn
}

func (p *clientPeer) startTransfer() {

}
