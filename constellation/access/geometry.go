package access

import (
	"math"
	"time"

	"github.com/joshuaferrara/go-satellite"
)

const (
	earthRadiusKm = 6378.137
	earthRate     = 7.2921150e-5
)

type vec3 [3]float64

func pointOf(latDeg, lonDeg float64) vec3 {
	phi := latDeg * math.Pi / 180
	lambda := lonDeg * math.Pi / 180
	return vec3{
		earthRadiusKm * math.Cos(phi) * math.Cos(lambda),
		earthRadiusKm * math.Cos(phi) * math.Sin(lambda),
		earthRadiusKm * math.Sin(phi),
	}
}

type look struct {
	alongKm float64
	squint  float64
	inc     float64
	side    string
	pass    string
}

// geometry is one satellite at one whole UTC second, seen from a ground point.
func geometry(sat satellite.Satellite, t time.Time, p vec3) look {
	r, v := earthFixed(sat, t)
	los := sub(r, p)
	cosi := dot(p, los) / (norm(p) * norm(los))
	inc := math.Acos(clamp(cosi, -1, 1)) * 180 / math.Pi

	h := cross(r, v)
	side := "RIGHT"
	if dot(h, p) > 0 {
		side = "LEFT"
	}

	phi := math.Atan2(r[2], math.Hypot(r[0], r[1]))
	lambda := math.Atan2(r[1], r[0])
	vNorth := -math.Sin(phi)*math.Cos(lambda)*v[0] - math.Sin(phi)*math.Sin(lambda)*v[1] + math.Cos(phi)*v[2]
	pass := "DESCENDING"
	if vNorth > 0 {
		pass = "ASCENDING"
	}

	ur := scale(r, 1/norm(r))
	horizontal := sub(v, scale(ur, dot(v, ur)))
	uAlong := scale(horizontal, 1/norm(horizontal))
	uRight := cross(scale(ur, -1), uAlong)
	toPoint := sub(p, r)
	along := dot(toPoint, uAlong)
	squint := math.Atan2(along, math.Abs(dot(toPoint, uRight))) * 180 / math.Pi

	return look{alongKm: along, squint: squint, inc: inc, side: side, pass: pass}
}

func earthFixed(sat satellite.Satellite, t time.Time) (vec3, vec3) {
	y, m, d := t.Date()
	hh, mm, ss := t.Clock()
	pos, vel := satellite.Propagate(sat, y, int(m), d, hh, mm, ss)
	theta := satellite.GSTimeFromDate(y, int(m), d, hh, mm, ss)
	c, s := math.Cos(theta), math.Sin(theta)
	r := vec3{pos.X*c + pos.Y*s, -pos.X*s + pos.Y*c, pos.Z}
	vr := vec3{vel.X*c + vel.Y*s, -vel.X*s + vel.Y*c, vel.Z}
	v := vec3{vr[0] + earthRate*r[1], vr[1] - earthRate*r[0], vr[2]}
	return r, v
}

func dot(a, b vec3) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func norm(a vec3) float64 { return math.Hypot(a[0], math.Hypot(a[1], a[2])) }

func sub(a, b vec3) vec3 { return vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

func scale(a vec3, k float64) vec3 { return vec3{a[0] * k, a[1] * k, a[2] * k} }

func cross(a, b vec3) vec3 {
	return vec3{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
