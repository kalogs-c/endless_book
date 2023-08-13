package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"

	"github.com/kalogs-c/endless_book/entities"
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
	case entities.Message:
		fmt.Println("Server received message")
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
	pid := s.ctx.SpawnChild(entities.NewUser(username, conn, s.ctx.PID()), username)

	s.sessions[pid.GetID()] = pid
}

func (s *Server) broadcast(sender *actor.PID, msg entities.Message) {
	for _, pid := range s.sessions {
		if !pid.Equals(sender) {
			log.Printf("Sending %v to %v\n", msg, pid)
			s.ctx.Send(pid, msg)
		}
	}
}
