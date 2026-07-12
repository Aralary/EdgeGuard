package main

import (
	"log"

	"github.com/aralary/edgeguard/internal/auth/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
