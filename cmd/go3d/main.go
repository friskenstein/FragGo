package main

import (
	"log"

	"github.com/johan/go3d/internal/game"
)

func main() {

	g, err := game.New()
	if err != nil {
		log.Fatal(err)
	}
	g.Run()
}
