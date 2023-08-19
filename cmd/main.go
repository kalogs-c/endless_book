package main

import (
	hollywood "github.com/anthdm/hollywood/actor"

	"github.com/kalogs-c/endless_book/actors"
)

func main() {
	engine := hollywood.NewEngine()
	engine.Spawn(actors.NewServer, "server")

	select {}
}
