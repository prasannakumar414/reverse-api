package main

import (
	"log"
	"os"

	httpserver "github.com/prasannakumar414/profile-retrieval-service/http"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := httpserver.NewServer(httpserver.Config{
		Addr: addr,
	})

	log.Printf("profile retrieval service listening on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
