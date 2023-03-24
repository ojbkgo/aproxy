package proxy

import (
	"context"
	"fmt"
	"io"
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
	Port       uint
	RemoteAddr string
	RemotePort uint
	proxyID    uint64
	ctl        net.Conn
	peers      map[uint64]*clientPeer
	mu         sync.Mutex
	localAddr  string
}

func (c *ClientManager) Dial(ctx context.Context) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.RemoteAddr, c.RemotePort))
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

func (c *ClientManager) runTrans(ctx context.Context, connID uint64) {
	if p, ok := c.peers[connID]; ok {
		p.startTransfer()
	}
}

func (c *ClientManager) DialTLS(ctx context.Context, addr string) error {
	return nil
}

func (c *ClientManager) connectLocal(ctx context.Context, connID uint64, localAddr string) error {
	if _, ok := c.peers[connID]; !ok {
		c.peers[connID] = &clientPeer{
			connID: connID,
		}
	}

	conn, err := net.Dial("tcp", localAddr)
	if err != nil {
		return err
	}

	c.peers[connID].b = conn
	return nil
}

func (c *ClientManager) registerDataChannel(ctx context.Context, proxyID, connID uint64) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.RemoteAddr, c.RemotePort+1))
	if err != nil {
		return err
	}

	_, err = createMessage(MessageTypeRegister, MessageDataChannelRegister{
		ProxyID: proxyID,
		ConnID:  connID,
	}).Write(conn)

	if err != nil {
		return err
	}

	rawMsg, err := readMessage(conn)
	if err != nil {
		return err
	}

	var ack *MessageDataChannelRegisterAck
	if v, ok := assertMessage(rawMsg).(*MessageDataChannelRegisterAck); !ok {
		return ErrBadConnection
	} else {
		ack = v
	}

	if ack.OK {
		c.peers[connID].a = conn
	}

	return nil
}

// todo connID 由服务端产生， 单边重启的时候，这个id值会不会重复
func (c *ClientManager) WaitConnection(ctx context.Context) error {

	for {
		rawMsg, err := readMessage(c.ctl)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		var revMsg *MessageRevConnect
		if v, ok := assertMessage(rawMsg).(*MessageRevConnect); !ok {
			fmt.Println("bad connection")
		} else {
			revMsg = v
		}

		err = c.connectLocal(ctx, revMsg.ConnID, revMsg.Address)
		if err != nil {
			fmt.Println(err.Error())

			_, _ = createMessage(MessageTypeRevConnectAck, &MessageRevConnectAck{
				OK:  false,
				Msg: err.Error(),
			}).Write(c.ctl)

			continue
		}

		err = c.registerDataChannel(ctx, revMsg.ProxyID, revMsg.ConnID)
		if err != nil {
			// todo ack false
			// todo remove from peers
			continue
		}

		c.runTrans(ctx, revMsg.ConnID)

	}

	return nil
}

type clientPeer struct {
	connID uint64
	a      net.Conn
	b      net.Conn
}

func (p *clientPeer) startTransfer() {
	go func() {
		for {
			n, err := io.Copy(p.a, p.b)
			if err != nil {
				fmt.Println(err)
			}

			fmt.Printf("from a to b: %d", n)
		}
	}()

	go func() {
		for {
			n, err := io.Copy(p.b, p.a)

			if err != nil {
				fmt.Println(err)
			}

			fmt.Printf("from b to a: %d", n)
		}
	}()
}
