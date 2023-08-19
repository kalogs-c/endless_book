package actors

import (
	"fmt"
	"log"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"

	"github.com/kalogs-c/endless_book/types"
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
	case types.Message:
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
	var word types.Message
	for {
		if err := u.conn.ReadJSON(&word); err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			return
		}

		word.Owner = u.Name
		word.CreatedAt = time.Now()

		if word.Content == "" {
			u.send(
				types.NewNotification(
					"Json message is invalid, must have a content field",
					"Server",
				),
			)
		}

		if len(word.Content) > 100 {
			u.send(
				types.NewNotification(
					"Json message is invalid, must be less than 100 characters",
					"Server",
				),
			)
		}

		if word.Type != "" {
			go u.handleWord(word)
		} else {
			u.send(types.NewNotification("Json message is invalid, must have a type field", "Server"))
		}
	}
}

func (u *User) handleWord(msg types.Message) {
	if msg := msg.ValidateWord(); msg != "" {
		u.send(types.NewNotification(msg, "Server"))
		return
	}

	if msg.IsQuiting() {
		u.ctx.Engine().Poison(u.ctx.PID())
		return
	}

	u.ctx.Send(u.serverPid, msg)
}

func (u *User) send(msg *types.Message) {
	var tmpl string

	if msg.Type == types.Word {
		tmpl = fmt.Sprintf(`
    <div id="messages" hx-swap-oob="beforeend" class="message">
      <span hidden>%s</span>
      <div>%s</div>
    </div>
  `, msg.Owner, msg.Content)
	} else if msg.Type == types.Notification {
		tmpl = fmt.Sprintf(`
    <div id="notifications" hx-swap-oob="beforeend" class="notification">
      <span hidden>%s</span>
      <div>%s</div>
    </div>
  `, msg.Owner, msg.Content)
	}
	fmt.Printf("sending html message: %v\n", tmpl)

	if err := u.conn.WriteMessage(websocket.TextMessage, []byte(tmpl)); err != nil {
		log.Printf("Error sending message: %v\n", err)
	}
}
