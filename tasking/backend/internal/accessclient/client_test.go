package accessclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aether/constellation/access"
	"aether/constellation/tasks"
)

func TestClientCreatesATask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if strings.Contains(string(raw), "unavailable") {
			t.Fatal("tasking sent a busy list")
		}
		if r.URL.Path != "/api/tasks" {
			t.Errorf("path %s", r.URL.Path)
		}
		task := sample()
		raw, _ = tasks.Marshal(task)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	got, err := New(srv.URL).Create(context.Background(), lisbon(), "order-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "task-1" || got.Status != "scheduled" || got.Offer.Satellite != "Aether-A1" {
		t.Fatalf("%+v", got)
	}
}

func TestQueryAndCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tasks/query":
			raw, _ := tasks.MarshalAll([]tasks.Task{sample()})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(raw)
		case "/api/tasks/task-1/cancel":
			task := sample()
			task.Status = "cancelled"
			raw, _ := tasks.Marshal(task)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(raw)
		default:
			t.Errorf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	client := New(srv.URL)
	list, err := client.Query(context.Background(), []string{"task-1"})
	if err != nil || len(list) != 1 || list[0].Status != "scheduled" {
		t.Fatalf("%+v %v", list, err)
	}
	cancelled, err := client.Cancel(context.Background(), "task-1")
	if err != nil || cancelled.Status != "cancelled" {
		t.Fatalf("%+v %v", cancelled, err)
	}
}

func lisbon() access.Request {
	return access.Request{
		Lat: 38.7223, Lon: -9.1393,
		WindowFrom: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC),
		WindowTo:   time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
		Mode:       "SPOTLIGHT", LookSide: "ANY", Pass: "ANY", Incidence: "performant",
	}
}

func sample() tasks.Task {
	start := time.Date(2026, 10, 29, 6, 6, 51, 0, time.UTC)
	return tasks.Task{
		ID: "task-1", ClientRef: "order-1", Status: "scheduled", CreatedAt: start,
		Lat: 38.7223, Lon: -9.1393, Mode: "SPOTLIGHT",
		RequestedLook: "ANY", RequestedPass: "ANY", IncidenceChoice: "performant",
		WindowStart: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
		Offer: access.Offer{
			Satellite: "Aether-A1", Plane: "Dawn", Class: "Standard",
			Start: start, End: start.Add(10 * time.Second),
			Duration: 10, Incidence: 26.12, LookSide: "RIGHT", Pass: "ASCENDING", Squint: 0.4,
		},
	}
}
