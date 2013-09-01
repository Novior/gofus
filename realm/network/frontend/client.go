package network

import (
	"bytes"
	"github.com/Blackrush/gofus/protocol/backend"
	protocol "github.com/Blackrush/gofus/protocol/frontend"
	_ "github.com/lib/pq"
	"io"
	"log"
	"net"
)

type Client interface {
	io.WriteCloser
	protocol.Sender
	protocol.CloseWither
	Alive() bool

	Id() uint64
	UserInfos() backend.UserInfos
	SetUserInfos(userInfos backend.UserInfos)
}

type net_client struct {
	net.Conn
	alive bool

	id        uint64
	userInfos backend.UserInfos
}

func new_net_client(conn net.Conn, id uint64) *net_client {
	return &net_client{
		Conn:  conn,
		alive: true,
		id:    id,
	}
}

func (client *net_client) Close() error {
	client.alive = false
	return client.Conn.Close()
}

func (client *net_client) Send(msg protocol.MessageContainer) (int, error) {
	log.Printf("[frontend-net-client-%04d] SND(%s) %+v", client.Id(), msg.Opcode(), msg)

	buf := new(bytes.Buffer)
	buf.WriteString(msg.Opcode())
	msg.Serialize(buf)
	buf.WriteString(output_msg_delimiter)

	n, err := buf.WriteTo(client)
	return int(n), err
}

func (client *net_client) CloseWith(msg protocol.MessageContainer) error {
	client.Send(msg)
	return client.Close()
}

func (client *net_client) Alive() bool {
	return client.alive
}

func (client *net_client) Id() uint64 {
	return client.id
}

func (client *net_client) UserInfos() backend.UserInfos {
	return client.userInfos
}

func (client *net_client) SetUserInfos(userInfos backend.UserInfos) {
	client.userInfos = userInfos
}