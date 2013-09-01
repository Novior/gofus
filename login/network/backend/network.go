package backend

import (
	"database/sql"
	"fmt"
	"github.com/Blackrush/gofus/protocol/backend"
	"github.com/Blackrush/gofus/shared"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

const (
	chunk_len = 32
	salt_len  = 100
)

var (
	message_delimiter = []byte{0}
)

type Configuration struct {
	Port uint16
}

type context struct {
	config Configuration

	db *sql.DB

	running        bool
	nextClientId   <-chan uint64
	nextClientSalt <-chan string
}

func New(database *sql.DB, config Configuration) shared.StartStopper {
	return &context{
		config: config,
		db:     database,
	}
}

func (ctx *context) Start() {
	if ctx.running {
		panic("backend network service already running")
	}
	ctx.running = true

	go client_id_generator(ctx)
	go client_salt_generator(ctx)
	go server_listen(ctx)

	log.Print("[backend-net] successfully started")
}

func (ctx *context) Stop() {
	ctx.running = false
}

func client_id_generator(ctx *context) {
	c := make(chan uint64)
	defer close(c)

	var nextId uint64

	ctx.nextClientId = c
	for ctx.running {
		nextId++
		c <- nextId
	}
}

func client_salt_generator(ctx *context) {
	c := make(chan string)
	defer close(c)

	src := rand.NewSource(time.Now().UnixNano())

	ctx.nextClientSalt = c
	for ctx.running {
		salt := shared.NextString(src, salt_len)
		c <- salt
	}
}

func server_listen(ctx *context) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", ctx.config.Port))
	if err != nil {
		panic(err.Error())
	}

	defer listener.Close()

	log.Printf("[backend-net] listening on %d", ctx.config.Port)

	for ctx.running {
		conn, err := listener.Accept()
		if err != nil {
			panic(err.Error())
		}

		go server_conn_rcv(ctx, conn)
	}
}

func conn_rcv(conn net.Conn) (backend.Message, bool) {
	var opcode uint16

	if n, err := backend.Read(conn, &opcode); n <= 0 || err == io.EOF {
		return nil, false
	} else if err != nil {
		panic(err.Error())
	}

	if msg, ok := backend.NewMsg(opcode); ok {
		if err := msg.Deserialize(conn); err == io.EOF {
			return nil, false
		} else if err != nil {
			panic(err.Error())
		}

		return msg, true
	}

	return nil, false
}

func server_conn_rcv(ctx *context, conn net.Conn) {
	client := &Client{
		WriteCloser: conn,
		id:          <-ctx.nextClientId,
		salt:        <-ctx.nextClientSalt,
		alive:       true,
	}
	defer server_conn_close(ctx, client)

	log.Printf("[backend-net-client-%04d] CONN", client.id)

	client.Send(&backend.HelloConnectMsg{client.salt})

	for ctx.running && client.alive {
		if msg, ok := conn_rcv(conn); ok {
			log.Printf("[backend-net-client-%04d] RCV(%d)", client.id, msg.Opcode())

			client_handle_data(ctx, client, msg)
		} else {
			break
		}
	}
}

func server_conn_close(ctx *context, client *Client) {
	client.Close()

	log.Printf("[backend-net-client-%04d] DCONN", client.id)
}

type Client struct {
	io.WriteCloser
	id    uint64
	salt  string
	alive bool
}

func (client *Client) Close() error {
	client.alive = false
	return client.WriteCloser.Close()
}

func (client *Client) Send(msg backend.Message) error {
	log.Printf("[backend-net-client-%04d] SND(%d)", client.id, msg.Opcode())

	backend.Put(client, msg.Opcode())
	return msg.Serialize(client)
}

func (client *Client) Id() uint64 {
	return client.id
}

func (client *Client) Salt() string {
	return client.salt
}

func (client *Client) Alive() bool {
	return client.alive
}