package main

import (
	"log"

	"github.com/aralary/edgeguard/internal/controlplane/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
