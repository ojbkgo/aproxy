package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

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

type Server struct {
	sig                 chan struct{}
	backends            map[uint64]*backend // port -> backend
	mu                  sync.Mutex
	ctlListener         net.Listener
	dataChannelListener net.Listener
	wg                  sync.WaitGroup
	//connID2Backend  map[uint64]int
}

func NewServer() *Server {
	return &Server{
		sig:      make(chan struct{}),
		backends: make(map[uint64]*backend),
		//connID2Backend:  make(map[uint64]int),
	}
}

func (m *Server) Wait() {
	time.Sleep(time.Second * 3)
	m.wg.Wait()
}

func (m *Server) Run(ctx context.Context, addr string) error {
	m.wg.Add(1)
	defer func() {
		m.wg.Done()
	}()

	lsn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	m.ctlListener = lsn

	for {
		conn, err := lsn.Accept()
		if err != nil {
			log.Println("wait for control channel connection over:", err.Error())
			return nil
		}

		err = m.Register(conn)
		if err != nil {
			log.Println("register proxy error:", err.Error())
			continue
		}
	}
}

func (m *Server) RunDataChannel(ctx context.Context, addr string) error {
	m.wg.Add(1)
	defer func() {
		m.wg.Done()
	}()

	fmt.Println(addr)
	lsn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	m.dataChannelListener = lsn
	fmt.Println("start listen...")
	for {
		conn, err := lsn.Accept()
		if err != nil {
			log.Println("wait data channel connection over: ", err.Error())
			return err
		}

		fmt.Println("start")
		msg, err := readMessage(conn)
		fmt.Println("end")
		if err != nil {
			log.Println(err.Error())
			return err
		}

		var dcRegisterMsg *MessageDataChannelRegister
		if v, ok := assertMessage(msg).(*MessageDataChannelRegister); !ok {
			log.Println("assertMessage MessageDataChannelRegister not exists")
			return ErrBadConnection
		} else {
			dcRegisterMsg = v
		}

		log.Println("receive data channel connection:", dcRegisterMsg.ProxyID, dcRegisterMsg.ConnID)

		err = m.Connect(ctx, dcRegisterMsg.ProxyID, dcRegisterMsg.ConnID, conn)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		_, err = createMessage(MessageTypeRegisterDataChannelAck, &MessageDataChannelRegisterAck{
			OK: true,
		}).Write(conn)

		log.Println("success send ack, start server side transfer...")
		m.startTransfer(ctx, dcRegisterMsg.ProxyID, dcRegisterMsg.ConnID)
	}
}

func (m *Server) Stats() {
	go func() {
		tk := time.NewTicker(time.Second * 3)
		for {
			select {
			case <-tk.C:
				if len(m.backends) > 0 {
					log.Printf("total backends: %d\n", len(m.backends))
					for _, it := range m.backends {
						log.Printf("backend: %s\t%s\tpeers: %d, proxyID: %d\n", it.App.Name, it.App.ServiceID, len(it.peers), it.id)
						for _, it2 := range it.peers {
							log.Printf("proxyID: %d\tpeer connID: %d\n", it.id, it2.connID)
						}
					}
				}
			case <-m.sig:
				return
			}
		}
	}()
}

func (m *Server) startTransfer(ctx context.Context, proxyID, connID uint64) {
	p := m.backends[proxyID].peers[connID]
	closed := utils.ForwardData(p.a, p.b)
	go func() {
		<-closed
		if _, ok := m.backends[proxyID]; ok {
			delete(m.backends[proxyID].peers, connID)
		}
	}()
}

func (m *Server) closeProxy(proxyID uint64) {
	if _, ok := m.backends[proxyID]; !ok {
		return
	}

	m.backends[proxyID].close()
	delete(m.backends, proxyID)
}

func (m *Server) waitProxyClose(conn net.Conn, proxyId uint64) {
	go func() {
		buf := make([]byte, 1)
		_, err := conn.Read(buf)
		if err == io.EOF {
			m.closeProxy(proxyId)
		}
	}()
}

func (m *Server) Register(conn net.Conn) error {
	rawMsg, err := readMessage(conn)
	if err != nil {
		return err
	}

	var regMsg *MessageRegister
	if v, ok := assertMessage(rawMsg).(*MessageRegister); !ok {
		log.Println("assertMessage MessageRegister not ok")
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

	log.Printf("proxy client [%d] registered, expose port %d\n", proxyID, regMsg.Port)

	err = m.backends[proxyID].waitConnection(context.Background())
	if err != nil {
		log.Printf("start server side proxy failed: %s, proxy id : %d\n", err.Error(), proxyID)

		msg := createMessage(MessageTypeRegisterAck, &MessageRegisterAck{
			ID:  0,
			Msg: err.Error(),
		})

		_, err = msg.Write(conn)
		if err != nil {
			return err
		}

		m.closeProxy(proxyID)

		return err
	}

	msg := createMessage(MessageTypeRegisterAck, &MessageRegisterAck{
		ID: proxyID,
	})

	_, err = msg.Write(conn)
	if err != nil {
		return err
	}

	m.waitProxyClose(conn, proxyID)

	return nil
}

func (m *Server) Connect(ctx context.Context, proxyID, connID uint64, conn net.Conn) error {
	if _, ok := m.backends[proxyID]; !ok {
		log.Println("backend not exists:", proxyID)
		return ErrBadConnection
	}

	if _, ok := m.backends[proxyID].peers[connID]; !ok {
		log.Println("backend peers not exists:", proxyID, connID)
		return ErrBadConnection
	}

	m.backends[proxyID].peers[connID].b = conn
	m.backends[proxyID].peers[connID].ready = true

	return nil
}

func (m *Server) Quit() {
	m.ctlListener.Close()
	m.dataChannelListener.Close()

	for _, it := range m.backends {
		it.close()
	}
}

type backend struct {
	id       uint64
	port     uint // 控制管道端口
	ctl      net.Conn
	listener net.Listener
	peers    map[uint64]*peer
	App      AppInfo
}

type peer struct {
	connID uint64
	a      net.Conn
	b      net.Conn
	ready  bool
}

func (b *backend) close() {
	b.listener.Close()
	b.ctl.Close()

	for _, it := range b.peers {
		if it.a != nil {
			it.a.Close()
		}

		if it.b != nil {
			it.b.Close()
		}
	}
}

func (b *backend) waitConnection(ctx context.Context) error {
	lsn, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", b.port))
	if err != nil {
		return err
	}
	b.listener = lsn

	// go func
	go func() {
		for {
			conn, err := lsn.Accept()
			if err != nil {
				log.Println("waiting for connect over: ", err.Error())
				return
			}

			err = b.reverseConnect(ctx, conn)
			if err != nil {
				log.Println("reverse connect error: ", err.Error())
				continue
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
	}).Write(b.ctl)

	if err != nil {
		return err
	}

	b.peers[connID] = &peer{
		connID: connID,
		a:      conn,
	}

	// read ack
	//msg, err := readMessage(b.ctl)
	//if err != nil {
	//	return err
	//}

	//var ackMsg *MessageRevConnectAck
	//if v, ok := assertMessage(msg).(*MessageRevConnectAck); !ok {
	//	return ErrBadConnection
	//} else {
	//	ackMsg = v
	//}

	//if !ackMsg.OK {
	//	// todo 清理消息
	//}

	return nil
}
