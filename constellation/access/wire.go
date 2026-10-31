package access

import (
	"encoding/json"
	"time"
)

// Encode builds the JSON body for an acquisition or task request.
func Encode(req Request, clientRef string) ([]byte, error) {
	body := struct {
		PointOfInterest struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"pointOfInterest"`
		AcquisitionWindow struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		} `json:"acquisitionWindow"`
		ImagingMode   string `json:"imagingMode"`
		LookSide      string `json:"lookSide"`
		PassDirection string `json:"passDirection"`
		Incidence     string `json:"incidence"`
		ClientRef     string `json:"clientRef,omitempty"`
	}{
		ImagingMode: req.Mode, LookSide: req.LookSide, PassDirection: req.Pass, Incidence: req.Incidence,
		ClientRef: clientRef,
	}
	body.PointOfInterest.Lat = req.Lat
	body.PointOfInterest.Lon = req.Lon
	body.AcquisitionWindow.Start = req.WindowFrom.UTC()
	body.AcquisitionWindow.End = req.WindowTo.UTC()
	return json.Marshal(body)
}

type whenJSON struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type resultJSON struct {
	Feasible        bool      `json:"feasible"`
	Satellite       string    `json:"satellite"`
	Plane           string    `json:"plane"`
	Class           string    `json:"class"`
	ImagingTime     *whenJSON `json:"imagingTime"`
	DurationSeconds int       `json:"durationSeconds"`
	IncidenceAngle  float64   `json:"incidenceAngle"`
	LookSide        string    `json:"lookSide"`
	PassDirection   string    `json:"passDirection"`
	SquintDeg       float64   `json:"squintDeg"`
}

// MarshalResult is the JSON an acquisition call returns.
func MarshalResult(offer Offer, found bool) ([]byte, error) {
	if !found {
		return []byte(`{"feasible":false}`), nil
	}
	body := resultJSON{
		Feasible: true, Satellite: offer.Satellite, Plane: offer.Plane, Class: offer.Class,
		ImagingTime:     &whenJSON{Start: offer.Start.UTC(), End: offer.End.UTC()},
		DurationSeconds: offer.Duration, IncidenceAngle: offer.Incidence,
		LookSide: offer.LookSide, PassDirection: offer.Pass, SquintDeg: offer.Squint,
	}
	return json.Marshal(body)
}

// UnmarshalResult reads an acquisition response.
func UnmarshalResult(raw []byte) (Offer, bool, error) {
	var body resultJSON
	if err := json.Unmarshal(raw, &body); err != nil {
		return Offer{}, false, err
	}
	if !body.Feasible {
		return Offer{}, false, nil
	}
	offer := Offer{
		Satellite: body.Satellite, Plane: body.Plane, Class: body.Class,
		Duration: body.DurationSeconds, Incidence: body.IncidenceAngle,
		LookSide: body.LookSide, Pass: body.PassDirection, Squint: body.SquintDeg,
	}
	if body.ImagingTime != nil {
		offer.Start = body.ImagingTime.Start.UTC()
		offer.End = body.ImagingTime.End.UTC()
	}
	return offer, true, nil
}
