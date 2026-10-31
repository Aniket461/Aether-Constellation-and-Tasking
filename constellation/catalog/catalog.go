package catalog

import (
	"database/sql"
	"fmt"
	"time"
)

// Plane is one orbit family.
type Plane struct {
	Name      string  `json:"name"`
	Color     string  `json:"color"`
	PeriodMin float64 `json:"periodMin"`
}

// Class is one bus, with the incidence it can steer to.
type Class struct {
	Name         string  `json:"name"`
	IncidenceMin float64 `json:"incidenceMin"`
	IncidenceMax float64 `json:"incidenceMax"`
}

// Mode is one way the fleet can take a picture.
type Mode struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Seconds    int      `json:"seconds"`
	RangeKm    float64  `json:"rangeKm"`
	AzimuthKm  float64  `json:"azimuthKm"`
	PerformMin float64  `json:"performMin"`
	PerformMax float64  `json:"performMax"`
	RequestMin float64  `json:"requestMin"`
	RequestMax float64  `json:"requestMax"`
	Classes    []string `json:"classes"`
}

// Satellite is one spacecraft and the TLE the propagator reads.
type Satellite struct {
	Name    string `json:"name"`
	Catalog int    `json:"catalog"`
	Plane   string `json:"plane"`
	Class   string `json:"class"`
	Look    string `json:"look"`
	Line1   string `json:"line1"`
	Line2   string `json:"line2"`
}

// Catalog is the fleet. Version changes when any of its tables change.
type Catalog struct {
	Version    int64       `json:"version"`
	Epoch      time.Time   `json:"epoch"`
	Planes     []Plane     `json:"planes"`
	Classes    []Class     `json:"classes"`
	Modes      []Mode      `json:"modes"`
	Satellites []Satellite `json:"satellites"`
}

// Load reads the fleet inserted by the migrations.
func Load(db *sql.DB) (Catalog, error) {
	var cat Catalog
	err := db.QueryRow(`SELECT version, epoch FROM constellation`).Scan(&cat.Version, &cat.Epoch)
	if err != nil {
		return Catalog{}, fmt.Errorf("constellation: %w", err)
	}
	cat.Epoch = cat.Epoch.UTC()

	planes, err := db.Query(`SELECT name, color, period_min FROM planes ORDER BY sort_order`)
	if err != nil {
		return Catalog{}, err
	}
	defer planes.Close()
	for planes.Next() {
		var plane Plane
		if err := planes.Scan(&plane.Name, &plane.Color, &plane.PeriodMin); err != nil {
			return Catalog{}, err
		}
		cat.Planes = append(cat.Planes, plane)
	}
	if err := planes.Err(); err != nil {
		return Catalog{}, err
	}

	classes, err := db.Query(`SELECT name, incidence_min, incidence_max FROM classes ORDER BY sort_order`)
	if err != nil {
		return Catalog{}, err
	}
	defer classes.Close()
	for classes.Next() {
		var class Class
		if err := classes.Scan(&class.Name, &class.IncidenceMin, &class.IncidenceMax); err != nil {
			return Catalog{}, err
		}
		cat.Classes = append(cat.Classes, class)
	}
	if err := classes.Err(); err != nil {
		return Catalog{}, err
	}

	modes, err := db.Query(`
		SELECT id, label, seconds, range_km, azimuth_km, perform_min, perform_max, request_min, request_max
		FROM imaging_modes ORDER BY sort_order`)
	if err != nil {
		return Catalog{}, err
	}
	defer modes.Close()
	for modes.Next() {
		var mode Mode
		if err := modes.Scan(
			&mode.ID, &mode.Label, &mode.Seconds, &mode.RangeKm, &mode.AzimuthKm,
			&mode.PerformMin, &mode.PerformMax, &mode.RequestMin, &mode.RequestMax,
		); err != nil {
			return Catalog{}, err
		}
		cat.Modes = append(cat.Modes, mode)
	}
	if err := modes.Err(); err != nil {
		return Catalog{}, err
	}

	links, err := db.Query(`SELECT mode, class FROM mode_classes ORDER BY mode, class`)
	if err != nil {
		return Catalog{}, err
	}
	defer links.Close()
	byMode := make(map[string][]string, len(cat.Modes))
	for links.Next() {
		var mode, class string
		if err := links.Scan(&mode, &class); err != nil {
			return Catalog{}, err
		}
		byMode[mode] = append(byMode[mode], class)
	}
	if err := links.Err(); err != nil {
		return Catalog{}, err
	}
	for i := range cat.Modes {
		cat.Modes[i].Classes = byMode[cat.Modes[i].ID]
	}

	sats, err := db.Query(`SELECT name, catalog, plane, class, look, line1, line2 FROM satellites ORDER BY sort_order`)
	if err != nil {
		return Catalog{}, err
	}
	defer sats.Close()
	for sats.Next() {
		var sat Satellite
		if err := sats.Scan(&sat.Name, &sat.Catalog, &sat.Plane, &sat.Class, &sat.Look, &sat.Line1, &sat.Line2); err != nil {
			return Catalog{}, err
		}
		cat.Satellites = append(cat.Satellites, sat)
	}
	if err := sats.Err(); err != nil {
		return Catalog{}, err
	}
	if len(cat.Satellites) == 0 || len(cat.Modes) == 0 {
		return Catalog{}, fmt.Errorf("constellation: catalog is empty")
	}
	return cat, nil
}
