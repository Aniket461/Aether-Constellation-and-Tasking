package access

import (
	"encoding/json"
	"io"
	"time"
)

const maxSearch = 7 * 24 * time.Hour

// Invalid is a request the acquisition endpoint refuses.
type Invalid struct {
	Msg string
}

func (e *Invalid) Error() string { return e.Msg }

// Parse reads an acquisition request. Booked intervals come from the task table.
func Parse(r io.Reader) (Request, error) {
	var body struct {
		PointOfInterest struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"pointOfInterest"`
		AcquisitionWindow struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"acquisitionWindow"`
		ImagingMode   string `json:"imagingMode"`
		LookSide      string `json:"lookSide"`
		PassDirection string `json:"passDirection"`
		Incidence     string `json:"incidence"`
	}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return Request{}, &Invalid{Msg: "request body must be JSON"}
	}
	if body.PointOfInterest.Lat < -90 || body.PointOfInterest.Lat > 90 || body.PointOfInterest.Lon < -180 || body.PointOfInterest.Lon > 180 {
		return Request{}, &Invalid{Msg: "point of interest is out of range"}
	}
	start, err := time.Parse(time.RFC3339Nano, body.AcquisitionWindow.Start)
	if err != nil {
		return Request{}, &Invalid{Msg: "window start must be RFC3339"}
	}
	end, err := time.Parse(time.RFC3339Nano, body.AcquisitionWindow.End)
	if err != nil {
		return Request{}, &Invalid{Msg: "window end must be RFC3339"}
	}
	if !end.After(start) || end.Sub(start) > maxSearch {
		return Request{}, &Invalid{Msg: "window must be at most 7 days"}
	}
	if body.ImagingMode == "" {
		return Request{}, &Invalid{Msg: "unknown imaging mode"}
	}
	if body.LookSide == "" {
		body.LookSide = "ANY"
	}
	if body.LookSide != "ANY" && body.LookSide != "LEFT" && body.LookSide != "RIGHT" {
		return Request{}, &Invalid{Msg: "look side must be ANY, LEFT, or RIGHT"}
	}
	if body.PassDirection == "" {
		body.PassDirection = "ANY"
	}
	if body.PassDirection != "ANY" && body.PassDirection != "ASCENDING" && body.PassDirection != "DESCENDING" {
		return Request{}, &Invalid{Msg: "pass direction must be ANY, ASCENDING, or DESCENDING"}
	}
	if body.Incidence == "" {
		body.Incidence = "performant"
	}
	if body.Incidence != "performant" && body.Incidence != "widest" {
		return Request{}, &Invalid{Msg: "incidence must be performant or widest"}
	}
	return Request{
		Lat: body.PointOfInterest.Lat, Lon: body.PointOfInterest.Lon,
		WindowFrom: start, WindowTo: end,
		Mode: body.ImagingMode, LookSide: body.LookSide, Pass: body.PassDirection,
		Incidence: body.Incidence,
	}, nil
}
