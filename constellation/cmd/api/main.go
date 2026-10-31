package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"aether/constellation/httpapi"
	"aether/constellation/tasks"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://aether:aether@127.0.0.1:5432/constellation?sslmode=disable"
	}
	delays, err := tasks.ParseDelays(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	api, err := httpapi.New(dsn, delays)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("task delays downlink %s processing %s", delays.Downlink, delays.Processing)
	go api.Advance(context.Background())
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: api.Handler(),
	}
	log.Printf("constellation listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
