package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"aether/constellation/access"
	"aether/tasking/internal/accessclient"
	"aether/tasking/internal/orders"
)

func (s *Server) feasibility(w http.ResponseWriter, r *http.Request) {
	req, ok := readRequest(w, r)
	if !ok {
		return
	}
	offer, found, err := s.orders.Earliest(r.Context(), req)
	if err != nil {
		var bad *accessclient.BadRequest
		if errors.As(err, &bad) || errors.Is(err, accessclient.ErrUnavailable) {
			writeAcquisitionError(w, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "could not search")
		return
	}
	if !found {
		writeJSON(w, map[string]any{"feasible": false})
		return
	}
	writeJSON(w, offerJSON(offer))
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	req, ok := readRequest(w, r)
	if !ok {
		return
	}
	order, err := s.orders.Place(r.Context(), req)
	if err != nil {
		var bad *accessclient.BadRequest
		if errors.As(err, &bad) || errors.Is(err, accessclient.ErrUnavailable) {
			writeAcquisitionError(w, err)
			return
		}
	}
	if errors.Is(err, orders.ErrNotFeasible) {
		writeError(w, http.StatusConflict, "not feasible")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not store the order")
		return
	}
	writeJSON(w, orderJSON(order))
}

func (s *Server) listOrders(w http.ResponseWriter, _ *http.Request) {
	list, err := s.orders.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read orders")
		return
	}
	body := make([]any, len(list))
	for i, order := range list {
		body[i] = orderJSON(order)
	}
	writeJSON(w, body)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	order, err := s.orders.Get(r.PathValue("id"))
	if errors.Is(err, orders.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read the order")
		return
	}
	writeJSON(w, orderJSON(order))
}

func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	order, err := s.orders.Cancel(r.Context(), r.PathValue("id"))
	if errors.Is(err, orders.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, orders.ErrFinished) {
		writeError(w, http.StatusConflict, "already finished")
		return
	}
	if errors.Is(err, accessclient.ErrUnavailable) {
		writeError(w, http.StatusServiceUnavailable, "constellation catalog is unavailable")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not cancel the order")
		return
	}
	writeJSON(w, orderJSON(order))
}

func readRequest(w http.ResponseWriter, r *http.Request) (access.Request, bool) {
	req, err := access.Parse(r.Body)
	if err != nil {
		msg := "request body must be JSON"
		var inv *access.Invalid
		if errors.As(err, &inv) {
			msg = inv.Msg
		}
		writeError(w, http.StatusBadRequest, msg)
		return access.Request{}, false
	}
	return req, true
}

func writeAcquisitionError(w http.ResponseWriter, err error) {
	var bad *accessclient.BadRequest
	if errors.As(err, &bad) {
		writeError(w, http.StatusBadRequest, bad.Message)
		return
	}
	writeError(w, http.StatusServiceUnavailable, "constellation catalog is unavailable")
}

func offerJSON(offer access.Offer) map[string]any {
	return map[string]any{
		"feasible":        true,
		"satellite":       offer.Satellite,
		"plane":           offer.Plane,
		"class":           offer.Class,
		"imagingTime":     spanJSON(offer.Start, offer.End),
		"durationSeconds": offer.Duration,
		"incidenceAngle":  offer.Incidence,
		"lookSide":        offer.LookSide,
		"passDirection":   offer.Pass,
		"squintDeg":       offer.Squint,
	}
}

func orderJSON(order orders.Order) map[string]any {
	var cancelled any
	if order.CancelledAt != nil {
		cancelled = order.CancelledAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"id":                order.ID,
		"taskId":            order.TaskID,
		"createdAt":         order.CreatedAt.UTC().Format(time.RFC3339),
		"cancelledAt":       cancelled,
		"status":            order.Status,
		"satellite":         order.Satellite,
		"plane":             order.Plane,
		"class":             order.Class,
		"pointOfInterest":   map[string]float64{"lat": order.Lat, "lon": order.Lon},
		"imagingMode":       order.Mode,
		"lookSide":          order.RequestedLook,
		"passDirection":     order.RequestedPass,
		"incidence":         order.IncidenceChoice,
		"acquisitionWindow": spanJSON(order.WindowStart, order.WindowEnd),
		"imagingTime":       spanJSON(order.ImagingStart, order.ImagingEnd),
		"durationSeconds":   order.Duration,
		"incidenceAngle":    order.Incidence,
		"computedLookSide":  order.LookSide,
		"computedPass":      order.Pass,
		"squintDeg":         order.Squint,
	}
}

func spanJSON(start, end time.Time) map[string]string {
	return map[string]string{
		"start": start.UTC().Format(time.RFC3339),
		"end":   end.UTC().Format(time.RFC3339),
	}
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
