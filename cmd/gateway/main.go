package main

import (
	"log"

	"github.com/aralary/edgeguard/internal/gateway/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
