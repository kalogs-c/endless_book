package entities

import (
	"fmt"
	"log"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"
)

type User struct {
	conn      *websocket.Conn
	ctx       *actor.Context
	serverPid *actor.PID
	Name      string
}

func NewUser(name string, conn *websocket.Conn, serverPid *actor.PID) actor.Producer {
	return func() actor.Receiver {
		return &User{
			Name:      name,
			conn:      conn,
			serverPid: serverPid,
		}
	}
}

func (u *User) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		u.ctx = ctx
		go u.listen()
		fmt.Printf("%s started\n", u.Name)
	case Message:
		log.Printf("%s sending %v\n", u.Name, msg)
		u.send(&msg)
	case actor.Stopped:
		_ = msg
		err := u.conn.Close()
		if err != nil {
			log.Printf("Error closing connection: %v\n", err)
		}
		fmt.Printf("%s stopped\n", u.Name)
	default:
		fmt.Printf("%s received %v\n", u.Name, msg)
	}
}

func (u *User) listen() {
	var word Message
	for {
		if err := u.conn.ReadJSON(&word); err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			return
		}

		word.Owner = u.Name
		word.CreatedAt = time.Now()

		if word.Type != "" {
			go u.handleWord(word)
		} else {
			u.send(NewNotification("Json message is invalid, must have a type field", "Server"))
		}
	}
}

func (u *User) handleWord(msg Message) {
	if msg := msg.ValidateWord(); msg != "" {
		u.send(NewNotification(msg, "Server"))
		return
	}

	if msg.IsQuiting() {
		u.ctx.Engine().Poison(u.ctx.PID())
		return
	}

	u.ctx.Send(u.serverPid, msg)
}

func (u *User) send(msg *Message) {
	fmt.Printf("sending json message: %v\n", msg)
	if err := u.conn.WriteJSON(msg); err != nil {
		fmt.Printf("Error writing message: %v\n", err)
		return
	}
}
