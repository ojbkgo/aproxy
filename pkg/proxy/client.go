package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/ojbkgo/aproxy/pkg/utils"
)

func NewClient() *Client {
	m := &Client{
		peers: make(map[uint64]*clientPeer),
	}

	return m
}

type AppInfo struct {
	Name      string // consul service name
	ServiceID string // consul service id
	Domain    string // nginx domain
	Host      string // host tags
}

type Client struct {
	proxyID              uint64
	ctl                  net.Conn
	peers                map[uint64]*clientPeer
	mu                   sync.Mutex
	localAddr            string
	serverAddr           string
	portA                int64
	portB                int64
	Info                 *AppInfo
	ReceiveFn            ClientHookReceiveConnection
	ResetFn              ClientHookConnectionReset
	BeforeConnectLocalFn ClientHookBeforeConnectLocal
	AfterConnectLocalFn  ClientHookAfterConnectLocal
	restart              chan struct{}
}

func (c *Client) Register(ctx context.Context, addr string, exposePort uint) error {
	ip, ports := utils.ParseAddr(addr)
	c.serverAddr = ip
	c.portA = ports[0]
	c.portB = ports[1]

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, c.portA))
	if err != nil {
		return err
	}

	msg := createMessage(MessageTypeRegister, &MessageRegister{
		Port: exposePort,

		App: c.Info,
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
		log.Println("assertMessage MessageRegisterAck not ok")
		return ErrBadConnection
	} else {
		ack = v
	}

	if ack.ID == 0 {
		return errors.New(ack.Msg)
	}

	c.proxyID = ack.ID
	c.ctl = conn

	return nil
}

func (c *Client) runTrans(ctx context.Context, connID uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.peers[connID]; ok {
		p.startTransfer()

		go func() {
			<-p.closed
			if c.ResetFn != nil {
				c.ResetFn(c.proxyID, connID)
			}
			c.mu.Lock()
			defer c.mu.Unlock()
			delete(c.peers, connID)
		}()
	}
}

func (c *Client) connectLocal(ctx context.Context, connID uint64, localAddr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.peers[connID]; !ok {
		c.peers[connID] = &clientPeer{
			connID: connID,
		}
	}

	conn, err := net.Dial("tcp", localAddr)
	if err != nil {
		if c.peers[connID].a != nil {
			c.peers[connID].a.Close()
		}

		if c.peers[connID].b != nil {
			c.peers[connID].b.Close()
		}
		delete(c.peers, connID)
		return err
	}

	c.peers[connID].b = conn
	return nil
}

func (c *Client) registerDataChannel(ctx context.Context, proxyID, connID uint64) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.serverAddr, c.portB))
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

	log.Println("register data channel success")

	rawMsg, err := readMessage(conn)
	if err != nil {
		return err
	}

	var ack *MessageDataChannelRegisterAck
	if v, ok := assertMessage(rawMsg).(*MessageDataChannelRegisterAck); !ok {
		log.Println("assertMessage MessageDataChannelRegisterAck not ok")
		return ErrBadConnection
	} else {
		ack = v
	}

	log.Println("get ack of register data channel success")
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.peers[connID]; !ok {
		conn.Close()
		return nil
	}

	if ack.OK {
		c.peers[connID].a = conn
	}

	return nil
}

func (c *Client) ProxyTo(ctx context.Context, addr string) error {
	for {
		rawMsg, err := readMessage(c.ctl)
		if err != nil {
			log.Printf("read rev connection error: %s\n", err.Error())
			c.restart <- struct{}{}
			return nil
		}

		if _, ok := assertMessage(rawMsg).(*MessageHeartbeat); ok {
			fmt.Println("heart beat, skip")
			continue
		}

		var revMsg *MessageRevConnect
		if v, ok := assertMessage(rawMsg).(*MessageRevConnect); !ok {
			log.Println("bad connection")
		} else {
			revMsg = v
		}

		log.Println("receive connection command:", addr, revMsg.ConnID, revMsg.ProxyID)
		if c.ReceiveFn != nil {
			c.ReceiveFn(c.proxyID, revMsg.ConnID)
		}

		err = c.connectLocal(ctx, revMsg.ConnID, addr)
		if err != nil {
			log.Printf("connect local error: %s\n", err.Error())

			//_, _ = createMessage(MessageTypeRevConnectAck, &MessageRevConnectAck{
			//	OK:  false,
			//	Msg: err.Error(),
			//}).Write(c.ctl)
		}

		//log.Println("connect local success:", addr)
		//_, err = createMessage(MessageTypeRevConnectAck, &MessageRevConnectAck{
		//	OK:  true,
		//	Msg: "OK",
		//}).Write(c.ctl)
		//if err != nil {
		//	log.Println("connect local success but write ack error: ", err.Error())
		//	continue
		//}

		err = c.registerDataChannel(ctx, revMsg.ProxyID, revMsg.ConnID)
		if err != nil {
			log.Println("register data channel error:", err.Error())
			continue
		}

		log.Println("registerDataChannel success")

		c.runTrans(ctx, revMsg.ConnID)
	}

	return nil
}

func (c *Client) Quit() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctl.Close()

	for _, it := range c.peers {
		if it.a != nil {
			it.a.Close()
		}

		if it.b == nil {
			it.b.Close()
		}
	}
}

type clientPeer struct {
	connID uint64
	a      net.Conn
	b      net.Conn
	closed chan struct{}
}

func (p *clientPeer) startTransfer() {
	p.closed = utils.ForwardData(p.a, p.b)
}
