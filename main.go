package main

import (
	"log"
	"nethackgo/game"
)

func main() {
	g := game.NewGame()
	if err := g.Run(); err != nil {
		log.Fatal(err)
	}
}