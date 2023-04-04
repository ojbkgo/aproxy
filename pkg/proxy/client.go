package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/ojbkgo/aproxy/pkg/utils"
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
		fmt.Println("assertMessage MessageRegisterAck not ok")
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
		fmt.Println("start transfer ", c.proxyID, connID)
		p.startTransfer()
	} else {
		fmt.Println("peers not ready ", connID)
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

	_, err = createMessage(MessageTypeRegisterDataChannel, MessageDataChannelRegister{
		ProxyID: proxyID,
		ConnID:  connID,
	}).Write(conn)

	if err != nil {
		return err
	}

	fmt.Println("register data channel success")

	rawMsg, err := readMessage(conn)
	if err != nil {
		return err
	}

	var ack *MessageDataChannelRegisterAck
	if v, ok := assertMessage(rawMsg).(*MessageDataChannelRegisterAck); !ok {
		fmt.Println("assertMessage MessageDataChannelRegisterAck not ok")
		return ErrBadConnection
	} else {
		ack = v
	}

	fmt.Println("get ack of register data channel success")

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

		fmt.Println("receive connection command:", revMsg.Address, revMsg.ConnID, revMsg.ProxyID)

		err = c.connectLocal(ctx, revMsg.ConnID, revMsg.Address)
		if err != nil {
			fmt.Println(err.Error())

			_, _ = createMessage(MessageTypeRevConnectAck, &MessageRevConnectAck{
				OK:  false,
				Msg: err.Error(),
			}).Write(c.ctl)

			continue
		}

		fmt.Println("connect local success:", revMsg.Address)

		_, err = createMessage(MessageTypeRevConnectAck, &MessageRevConnectAck{
			OK:  true,
			Msg: "OK",
		}).Write(c.ctl)
		if err != nil {
			fmt.Println("bad connection:", "write ack error", err.Error())
			continue
		}

		err = c.registerDataChannel(ctx, revMsg.ProxyID, revMsg.ConnID)

		if err != nil {
			fmt.Println("registerDataChannel error", err.Error())
			// todo ack false
			// todo remove from peers
			continue
		}

		fmt.Println("registerDataChannel success")

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
	utils.ForwardData(p.a, p.b)
}
