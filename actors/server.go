package actors

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/kalogs-c/endless_book/connections"
	"github.com/kalogs-c/endless_book/types"
)

type Server struct {
	ctx            *actor.Context
	db             *mongo.Client
	sessions       map[string]*actor.PID
	messagesBuffer []*types.Message
}

func NewServer() actor.Receiver {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := connections.NewMongoDBConnection(ctx, "mongodb://root:root@localhost:27017")
	if err != nil {
		panic(err)
	}

	return &Server{
		sessions:       make(map[string]*actor.PID),
		messagesBuffer: []*types.Message{},
		db:             client,
	}
}

func (s *Server) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		fmt.Println("Server started on port 8080")
		s.serve()
		s.ctx = ctx
		_ = msg
	case types.Message:
		fmt.Println("Server received message")
		s.saveMessage(msg)
		s.broadcast(msg)
	default:
		fmt.Printf("Server received %v\n", msg)
	}
}

func (s *Server) serve() {
	go func() {
		http.HandleFunc("/", s.handleIndex)
		http.HandleFunc("/ws", s.handleWebsocket)
		http.HandleFunc("/connect", s.handleNewConnection)
		http.ListenAndServe(":8080", nil)
	}()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	tmpl.Execute(w, nil)
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

	if len(s.messagesBuffer) > 0 {
		// conn.WriteJSON(s.messagesBuffer)
	} else {
		cursor, err := s.db.Database("endless_book").Collection("messages").Find(context.Background(), nil)
		if err == nil {
			for cursor.Next(context.Background()) {
				var msg types.Message
				err := cursor.Decode(&msg)
				if err != nil {
					log.Fatal(err)
				}
				s.broadcast(msg)
			}
		}
	}

	fmt.Printf("Sessions: %+v\n", s.sessions)

	// s.broadcast(*types.NewNotification(fmt.Sprintf("%s joined the chat", username), "Server"))
	s.sessions[pid.GetID()] = pid
}

func (s *Server) handleNewConnection(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.Form.Get("username")
	log.Printf("New connection: %s\n", username)

	tmpl := template.Must(template.ParseFiles("templates/components/connection_form.html"))

	tmpl.ExecuteTemplate(w, "connection_form.html", struct{ Username string }{username})
}

func (s *Server) broadcast(msg types.Message) {
	for _, pid := range s.sessions {
		log.Printf("Sending %v to %v\n", msg, pid)
		s.ctx.Send(pid, msg)
	}
}

func (s *Server) saveMessage(msg types.Message) {
	fmt.Printf("Saving message, BUF size: %d\n", len(s.messagesBuffer))
	s.messagesBuffer = append(s.messagesBuffer, &msg)

	if len(s.messagesBuffer) > 10000 {
		documents := make([]interface{}, len(s.messagesBuffer))
		for i, v := range s.messagesBuffer {
			documents[i] = v
		}

		log.Printf("Saving %d messages\n", len(documents))
		_, err := s.db.Database("endless_book").
			Collection("messages").
			InsertMany(context.Background(), documents)
		if err != nil {
			log.Fatal(err)
		} else {
			s.messagesBuffer = []*types.Message{}
		}
	}
}
