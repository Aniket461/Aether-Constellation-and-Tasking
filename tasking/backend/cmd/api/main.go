package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"aether/tasking/internal/httpapi"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://aether:aether@127.0.0.1:5432/tasking?sslmode=disable"
	}
	constellationURL := os.Getenv("CONSTELLATION_URL")
	if constellationURL == "" {
		constellationURL = "http://127.0.0.1:8081"
	}
	api, err := httpapi.New(dsn, constellationURL)
	if err != nil {
		log.Fatal(err)
	}
	go api.Sync(context.Background())
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: api.Handler(),
	}
	log.Printf("tasking listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
