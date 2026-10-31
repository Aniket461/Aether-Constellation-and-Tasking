package propagate

import (
	"math"
	"time"

	"github.com/joshuaferrara/go-satellite"

	"aether/constellation/catalog"
)

// Sample is one Earth-fixed position of a satellite.
type Sample struct {
	Time  time.Time
	Lat   float64
	Lon   float64
	AltKm float64
	EcefX float64
	EcefY float64
	EcefZ float64
}

// Track is the sampled arc of one satellite.
type Track struct {
	Name    string
	Samples []Sample
}

type prepared struct {
	name string
	sat  satellite.Satellite
}

// Propagator holds parsed TLEs.
type Propagator struct {
	sats []prepared
}

// New parses every TLE once.
func New(sats []catalog.Satellite) *Propagator {
	out := make([]prepared, len(sats))
	for i, s := range sats {
		out[i] = prepared{
			name: s.Name,
			sat:  satellite.TLEToSat(s.Line1, s.Line2, satellite.GravityWGS84),
		}
	}
	return &Propagator{sats: out}
}

// Ephemeris samples each satellite from from through to, inclusive, on whole-second steps.
func (p *Propagator) Ephemeris(from, to time.Time, step time.Duration) []Track {
	from = from.UTC().Truncate(time.Second)
	to = to.UTC().Truncate(time.Second)
	if step < time.Second {
		step = time.Second
	}
	step = step.Truncate(time.Second)
	tracks := make([]Track, len(p.sats))
	for i, s := range p.sats {
		samples := make([]Sample, 0, int(to.Sub(from)/step)+1)
		for t := from; !t.After(to); t = t.Add(step) {
			samples = append(samples, at(s.sat, t))
		}
		tracks[i] = Track{Name: s.name, Samples: samples}
	}
	return tracks
}

func at(sat satellite.Satellite, t time.Time) Sample {
	y, m, d := t.Date()
	hh, mm, ss := t.Clock()
	pos, _ := satellite.Propagate(sat, y, int(m), d, hh, mm, ss)
	gmst := satellite.GSTimeFromDate(y, int(m), d, hh, mm, ss)
	ecef := satellite.ECIToECEF(pos, gmst)
	alt, _, ll := satellite.ECIToLLA(pos, gmst)
	return Sample{
		Time:  t,
		Lat:   ll.Latitude * 180 / math.Pi,
		Lon:   normLon(ll.Longitude * 180 / math.Pi),
		AltKm: alt,
		EcefX: ecef.X,
		EcefY: ecef.Y,
		EcefZ: ecef.Z,
	}
}

func normLon(deg float64) float64 {
	lon := math.Mod(deg, 360)
	if lon > 180 {
		lon -= 360
	} else if lon <= -180 {
		lon += 360
	}
	return lon
}
