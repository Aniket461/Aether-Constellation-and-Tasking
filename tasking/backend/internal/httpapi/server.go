package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"aether/tasking/internal/accessclient"
	"aether/tasking/internal/orders"
)

// Server serves feasibility and orders.
type Server struct {
	orders *orders.Store
}

// New migrates the tasking database. Acquisition math is asked of the constellation.
func New(dsn, constellationURL string) (*Server, error) {
	store, err := orders.Open(dsn, accessclient.New(constellationURL))
	if err != nil {
		return nil, err
	}
	return &Server{orders: store}, nil
}

// Sync copies constellation task status onto open orders once a second.
func (s *Server) Sync(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.orders.Sync(context.Background()); err != nil {
				log.Printf("task status sync: %v", err)
			}
		}
	}
}

// Handler is the HTTP API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/feasibility", s.feasibility)
	mux.HandleFunc("POST /api/orders", s.createOrder)
	mux.HandleFunc("GET /api/orders", s.listOrders)
	mux.HandleFunc("GET /api/orders/{id}", s.getOrder)
	mux.HandleFunc("POST /api/orders/{id}/cancel", s.cancelOrder)
	return mux
}
