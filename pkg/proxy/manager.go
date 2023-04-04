package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/ojbkgo/aproxy/pkg/utils"
)

var (
	maxProxyID = uint64(0)
	maxConnID  = uint64(1000)
)

func nextProxyID() uint64 {
	maxProxyID++
	return maxProxyID
}

func nextConnID() uint64 {
	maxConnID++
	return maxConnID
}

type Manager struct {
	proxyID  uint64
	connID   uint64
	sig      chan struct{}
	backends map[uint64]*backend // port -> backend
	mu       sync.Mutex
	//connID2Backend  map[uint64]int
}

func NewManager() *Manager {
	return &Manager{
		sig:      make(chan struct{}),
		backends: make(map[uint64]*backend),
		//connID2Backend:  make(map[uint64]int),
	}
}

func (m *Manager) Run(ctx context.Context, addr string) error {
	defer func() {
		m.Stop()
	}()

	lsn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := lsn.Accept()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		fmt.Println("register...")
		err = m.Register(conn)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
	}
}

func (m *Manager) RunDataChannel(ctx context.Context, addr string) error {
	defer func() {
		m.Stop()
	}()

	lsn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for {
		conn, err := lsn.Accept()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		msg, err := readMessage(conn)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		var dcRegisterMsg *MessageDataChannelRegister
		if v, ok := assertMessage(msg).(*MessageDataChannelRegister); !ok {
			fmt.Println("assertMessage MessageDataChannelRegister not exists")
			return ErrBadConnection
		} else {
			dcRegisterMsg = v
		}

		fmt.Println("receive data channel connection:", dcRegisterMsg.ProxyID, dcRegisterMsg.ConnID)

		err = m.Connect(ctx, dcRegisterMsg.ProxyID, dcRegisterMsg.ConnID, conn)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		_, err = createMessage(MessageTypeRegisterDataChannelAck, &MessageDataChannelRegisterAck{
			OK: true,
		}).Write(conn)

		fmt.Println("success send ack, start server side transfer...")
		m.startTransfer(ctx, dcRegisterMsg.ProxyID, dcRegisterMsg.ConnID)
	}
}

func (m *Manager) startTransfer(ctx context.Context, proxyID, connID uint64) {
	p := m.backends[proxyID].peers[connID]
	utils.ForwardData(p.a, p.b)
}

func (m *Manager) Register(conn net.Conn) error {
	fmt.Println("read message")
	rawMsg, err := readMessage(conn)
	if err != nil {
		return err
	}

	var regMsg *MessageRegister
	if v, ok := assertMessage(rawMsg).(*MessageRegister); !ok {
		fmt.Println("assertMessage MessageRegister not exists")
		return ErrBadConnection
	} else {
		regMsg = v
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	proxyID := nextProxyID()
	m.backends[proxyID] = &backend{
		ctl:   conn,
		id:    proxyID,
		port:  regMsg.Port,
		peers: make(map[uint64]*peer),
	}

	fmt.Println("register...", regMsg.Port, proxyID)

	err = m.backends[proxyID].waitConnection(context.Background())
	if err != nil {
		fmt.Println(err.Error())
	}

	msg := createMessage(MessageTypeRegisterAck, &MessageRegisterAck{
		ID: proxyID,
	})

	_, err = msg.Write(conn)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) Connect(ctx context.Context, proxyID, connID uint64, conn net.Conn) error {
	if _, ok := m.backends[proxyID]; !ok {
		fmt.Println("backend not exists:", proxyID)
		return ErrBadConnection
	}

	if _, ok := m.backends[proxyID].peers[connID]; !ok {
		fmt.Println("backend peers not exists:", proxyID, connID)
		return ErrBadConnection
	}

	m.backends[proxyID].peers[connID].b = conn
	m.backends[proxyID].peers[connID].ready = true

	return nil
}

func (m *Manager) Stop() {

}

type backend struct {
	id    uint64
	port  uint // 控制管道端口
	ctl   net.Conn
	peers map[uint64]*peer
}

type peer struct {
	connID uint64
	a      net.Conn
	b      net.Conn
	ready  bool
}

func (b *backend) waitConnection(ctx context.Context) error {
	lsn, err := net.Listen("tcp", fmt.Sprintf(":%d", b.port))
	if err != nil {
		return err
	}

	// go func
	go func() {
		for {
			conn, err := lsn.Accept()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}

			err = b.reverseConnect(ctx, conn)
			if err != nil {
				fmt.Println(err.Error())
				break
			}
		}
	}()

	return nil
}

func (b *backend) reverseConnect(ctx context.Context, conn net.Conn) error {
	connID := nextConnID()

	_, err := createMessage(MessageTypeRevConnect, &MessageRevConnect{
		ProxyID: b.id,
		ConnID:  connID,
		Address: fmt.Sprintf(":%d", b.port+1),
	}).Write(b.ctl)

	if err != nil {
		return err
	}

	b.peers[connID] = &peer{
		connID: connID,
		a:      conn,
	}

	// read ack
	msg, err := readMessage(b.ctl)
	if err != nil {
		fmt.Println("reverseConnect readMessage error:", err.Error())
		return err
	}

	var ackMsg *MessageRevConnectAck
	if v, ok := assertMessage(msg).(*MessageRevConnectAck); !ok {
		fmt.Println("assertMessage MessageRevConnectAck not exists")
		return ErrBadConnection
	} else {
		ackMsg = v
	}

	if !ackMsg.OK {
		// todo 清理消息
	}

	return nil
}
