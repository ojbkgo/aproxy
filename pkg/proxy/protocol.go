package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const MessagePrefix = 0xfe

const (
	MessageTypeRegisterID uint32 = iota + 1
	MessageTypeRegister
	MessageTypeRegisterAck
	MessageTypeRevConnect
	MessageTypeRevConnectAck
	MessageTypeRegisterDataChannel
	MessageTypeRegisterDataChannelAck
	MessageTypeHeartbeat
)

const (
	SizeHeader = 9
)

var (
	ErrBadConnection    = errors.New("bad connection")
	ErrPortNotAvailable = errors.New("no available port")
)

type MessageInterface interface {
	Marshal() []byte
	UnMarshal(data []byte) error
}

type Message struct {
	size uint32
	typ  uint32
	data []byte
}

func (m *Message) Read(reader io.Reader) (int, error) {
	header := make([]byte, SizeHeader)
	size, err := reader.Read(header)
	if err != nil {
		return 0, errors.New("i/o read error: " + err.Error())
	}

	if size != SizeHeader {
		return 0, ErrBadConnection
	}

	if header[0] != MessagePrefix {
		return 0, ErrBadConnection
	}

	m.typ = binary.BigEndian.Uint32(header[1:5])
	m.size = binary.BigEndian.Uint32(header[5:]) - SizeHeader

	fmt.Println(m.typ, m.size)

	m.data = make([]byte, m.size)
	size, err = reader.Read(m.data)
	if err != nil {
		return 0, err
	}
	fmt.Println(len(m.data), size)
	if size != int(m.size) {
		return 0, ErrBadConnection
	}

	return size, nil
}

func (m *Message) Write(writer io.Writer) (int, error) {
	buf := []byte{MessagePrefix}

	buf = binary.BigEndian.AppendUint32(buf, m.typ)
	buf = binary.BigEndian.AppendUint32(buf, m.size)
	buf = append(buf, m.data...)

	return writer.Write(buf)
}

func (m *Message) Type() uint32 {
	return m.typ
}

func (m *Message) Data() []byte {
	return m.data
}

func (m *Message) SetData(typ uint32, data []byte) {
	m.typ = typ
	m.data = data
	m.size = uint32(len(data) + SizeHeader)
}
