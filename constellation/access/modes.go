package access

// modeSpec is one imaging mode from the catalog.
type modeSpec struct {
	id         string
	seconds    int
	performMin float64
	performMax float64
	requestMin float64
	requestMax float64
	classes    map[string]bool
}

// band is the incidence interval for this mode, choice, and bus reach.
// choice "widest" uses the mode's requestable band. Anything else uses performant.
func band(mode modeSpec, className, choice string, reachMin, reachMax float64, hasReach bool) (float64, float64, bool) {
	if !mode.classes[className] || !hasReach {
		return 0, 0, false
	}
	lo, hi := mode.performMin, mode.performMax
	if choice == "widest" {
		lo, hi = mode.requestMin, mode.requestMax
	}
	if reachMin > lo {
		lo = reachMin
	}
	if reachMax < hi {
		hi = reachMax
	}
	if lo > hi {
		return 0, 0, false
	}
	return lo, hi, true
}
