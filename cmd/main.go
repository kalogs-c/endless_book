package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"
)

type Server struct {
	ctx      *actor.Context
	sessions map[string]*actor.PID
}

func NewServer() actor.Receiver {
	return &Server{
		sessions: make(map[string]*actor.PID),
	}
}

func (s *Server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("Server started on port 8080")
		s.serve()
		s.ctx = ctx
		_ = msg
	case Word:
		fmt.Println("Server received word")
		s.broadcast(ctx.Sender(), msg)
	default:
		fmt.Printf("Server received %v\n", msg)
	}
}

func (s *Server) serve() {
	go func() {
		http.HandleFunc("/ws", s.handleWebsocket)
		http.ListenAndServe(":8080", nil)
	}()
}

func (s *Server) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New connection")
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	username := r.URL.Query().Get("username")
	pid := s.ctx.SpawnChild(NewUser(username, conn, s.ctx.PID()), username)

	s.sessions[pid.GetID()] = pid
}

func (s *Server) broadcast(sender *actor.PID, msg Word) {
	for _, pid := range s.sessions {
		if !pid.Equals(sender) {
			log.Printf("Sending %v to %v\n", msg, pid)
			s.ctx.Send(pid, msg)
		}
	}
}

type Word struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
	Owner   string `json:"owner"`
}

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
	case Word:
		log.Printf("%s sending %v\n", u.Name, msg)
		u.send(&msg)
	case actor.Stopped:
		fmt.Printf("%s stopped\n", u.Name)
		u.conn.Close()
	default:
		fmt.Printf("%s received %v\n", u.Name, msg)
	}
}

func (u *User) listen() {
	var word Word
	for {
		if err := u.conn.ReadJSON(&word); err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			return
		}

		go u.handleWord(word)
	}
}

func (u *User) handleWord(word Word) {
	if len(strings.Split(word.Content, " ")) != 1 {
		u.send(&Word{
			ID:      0,
			Content: "Invalid content, must be a single word",
			Owner:   u.Name,
		})
		return
	}

	switch word.Content {
	case "exit":
		u.ctx.Engine().Poison(u.ctx.PID())
	default:
		u.ctx.Send(u.serverPid, word)
	}
}

func (u *User) send(word *Word) {
	fmt.Printf("sending json message: %v\n", word)
	if err := u.conn.WriteJSON(word); err != nil {
		fmt.Printf("Error writing message: %v\n", err)
		return
	}
}

func main() {
	engine := actor.NewEngine()
	engine.Spawn(NewServer, "server")

	select {}
}
