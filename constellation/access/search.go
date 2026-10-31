package access

import (
	"time"

	"github.com/joshuaferrara/go-satellite"

	"aether/constellation/catalog"
)

// Request is one feasibility question.
type Request struct {
	Lat        float64
	Lon        float64
	WindowFrom time.Time
	WindowTo   time.Time
	Mode       string
	LookSide   string
	Pass       string
	Incidence  string
}

// Offer is the earliest legal collection.
type Offer struct {
	Satellite string
	Plane     string
	Class     string
	Start     time.Time
	End       time.Time
	Duration  int
	Incidence float64
	LookSide  string
	Pass      string
	Squint    float64
}

// Span is a half-open busy interval on one satellite.
type Span struct {
	Satellite string
	Start     time.Time
	End       time.Time
}

type craft struct {
	name  string
	plane string
	class string
	look  string
	sat   satellite.Satellite
}

// Searcher walks the acquisition window for the earliest broadside second.
type Searcher struct {
	craft []craft
	reach map[string][2]float64
	modes map[string]modeSpec
}

// New parses the catalog loaded from Postgres.
func New(cat catalog.Catalog) *Searcher {
	out := make([]craft, len(cat.Satellites))
	for i, s := range cat.Satellites {
		out[i] = craft{
			name: s.Name, plane: s.Plane, class: s.Class, look: s.Look,
			sat: satellite.TLEToSat(s.Line1, s.Line2, satellite.GravityWGS84),
		}
	}
	reach := make(map[string][2]float64, len(cat.Classes))
	for _, class := range cat.Classes {
		reach[class.Name] = [2]float64{class.IncidenceMin, class.IncidenceMax}
	}
	modes := make(map[string]modeSpec, len(cat.Modes))
	for _, mode := range cat.Modes {
		spec := modeSpec{
			id: mode.ID, seconds: mode.Seconds,
			performMin: mode.PerformMin, performMax: mode.PerformMax,
			requestMin: mode.RequestMin, requestMax: mode.RequestMax,
			classes: make(map[string]bool, len(mode.Classes)),
		}
		for _, name := range mode.Classes {
			spec.classes[name] = true
		}
		modes[mode.ID] = spec
	}
	return &Searcher{craft: out, reach: reach, modes: modes}
}

// Knows reports whether the catalog includes this imaging mode.
func (s *Searcher) Knows(mode string) bool {
	_, ok := s.modes[mode]
	return ok
}

// Earliest returns the soonest collection that passes every test.
// busy intervals are already reserved and are skipped.
func (s *Searcher) Earliest(req Request, busy []Span) (Offer, bool) {
	mode, ok := s.modes[req.Mode]
	if !ok || !req.WindowTo.After(req.WindowFrom) {
		return Offer{}, false
	}
	p := pointOf(req.Lat, req.Lon)
	var best Offer
	found := false
	for _, c := range s.craft {
		reach, hasReach := s.reach[c.class]
		lo, hi, ok := band(mode, c.class, req.Incidence, reach[0], reach[1], hasReach)
		if !ok {
			continue
		}
		if !rosterCan(c.look, req.LookSide) {
			continue
		}
		offer, ok := earliestOn(c, mode, req, p, lo, hi, busy)
		if !ok {
			continue
		}
		if !found || offer.Start.Before(best.Start) {
			best = offer
			found = true
		}
	}
	return best, found
}

func rosterCan(roster, requested string) bool {
	if requested == "" || requested == "ANY" {
		return true
	}
	if roster == "either" {
		return true
	}
	return roster == "right" && requested == "RIGHT" || roster == "left" && requested == "LEFT"
}

func earliestOn(c craft, mode modeSpec, req Request, p vec3, incMin, incMax float64, busy []Span) (Offer, bool) {
	const step = 20 * time.Second
	start := req.WindowFrom.UTC().Truncate(time.Second)
	end := req.WindowTo.UTC().Truncate(time.Second)
	var prevTime time.Time
	var prevAlong float64
	havePrev := false
	for t := start; !t.After(end); t = t.Add(step) {
		g := geometry(c.sat, t, p)
		if havePrev && prevAlong*g.alongKm < 0 {
			if offer, ok := refine(c, mode, req, p, prevTime, t, incMin, incMax, busy); ok {
				return offer, true
			}
		}
		prevTime = t
		prevAlong = g.alongKm
		havePrev = true
	}
	return Offer{}, false
}

func refine(c craft, mode modeSpec, req Request, p vec3, from, to time.Time, incMin, incMax float64, busy []Span) (Offer, bool) {
	var prev look
	have := false
	for t := from; !t.After(to); t = t.Add(time.Second) {
		g := geometry(c.sat, t, p)
		if have && prev.alongKm*g.alongKm < 0 {
			chosenT, chosen := t, g
			if abs(prev.squint) < abs(g.squint) {
				chosenT, chosen = t.Add(-time.Second), prev
			}
			if abs(chosen.squint) <= 1 && accepts(c, mode, req, chosen, chosenT, incMin, incMax, busy) {
				return makeOffer(c, mode, chosenT, chosen), true
			}
			return Offer{}, false
		}
		prev = g
		have = true
	}
	return Offer{}, false
}

func accepts(c craft, mode modeSpec, req Request, g look, start time.Time, incMin, incMax float64, busy []Span) bool {
	if g.inc < incMin || g.inc > incMax {
		return false
	}
	if !sideOK(c.look, req.LookSide, g.side) {
		return false
	}
	if req.Pass != "" && req.Pass != "ANY" && req.Pass != g.pass {
		return false
	}
	finish := start.Add(time.Duration(mode.seconds) * time.Second)
	if start.Before(req.WindowFrom) || finish.After(req.WindowTo) {
		return false
	}
	for _, span := range busy {
		if span.Satellite == c.name && span.Start.Before(finish) && span.End.After(start) {
			return false
		}
	}
	return true
}

func sideOK(roster, requested, computed string) bool {
	if roster != "either" {
		want := "RIGHT"
		if roster == "left" {
			want = "LEFT"
		}
		if computed != want {
			return false
		}
	}
	if requested == "" || requested == "ANY" {
		return true
	}
	return requested == computed
}

func makeOffer(c craft, mode modeSpec, start time.Time, g look) Offer {
	return Offer{
		Satellite: c.name,
		Plane:     c.plane,
		Class:     c.class,
		Start:     start,
		End:       start.Add(time.Duration(mode.seconds) * time.Second),
		Duration:  mode.seconds,
		Incidence: g.inc,
		LookSide:  g.side,
		Pass:      g.pass,
		Squint:    g.squint,
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
