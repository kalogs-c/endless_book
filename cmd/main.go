package main

import (
	"github.com/anthdm/hollywood/actor"

	"github.com/kalogs-c/endless_book/server"
)

func main() {
	engine := actor.NewEngine()
	engine.Spawn(server.NewServer, "server")

	select {}
}
