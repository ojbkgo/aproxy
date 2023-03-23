package proxy

import (
	"encoding/json"
	"io"
)

type MessageDataChannelRegister struct {
	ProxyID uint64
	ConnID  uint64
}

type MessageRegister struct {
	Port uint
}

type MessageRegisterAck struct {
	ID uint64
}

type MessageRevConnect struct {
	ProxyID uint64
	ConnID  uint64
	Address string
}

type MessageRevConnectAck struct {
	OK  bool
	Msg string
}

func readMessage(reader io.Reader) (*Message, error) {
	msg := &Message{}
	_, err := msg.Read(reader)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func assertMessage(m *Message) interface{} {
	var o interface{}

	switch m.Type() {
	case MessageTypeRegister:
		o = &MessageRegister{}
	case MessageTypeRegisterAck:
		o = &MessageRegisterAck{}
	case MessageTypeRevConnect:
		o = &MessageRevConnect{}
	case MessageTypeRevConnectAck:
		o = &MessageRevConnectAck{}
	}

	_ = json.Unmarshal(m.Data(), o)
	return o
}

func createMessage(typ uint32, i interface{}) *Message {
	msg := &Message{}
	b, _ := json.Marshal(i)
	msg.SetData(typ, b)
	return msg
}
