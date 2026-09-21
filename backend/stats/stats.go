package stats

import "slices"

func P95(window []uint64) uint64 {
	sorted := make([]uint64, len(window))
	copy(sorted, window)
	slices.Sort(sorted)
	index_of_95 := int(float64(len(window)) * 0.95)
	value := sorted[index_of_95]
	return value
}

func ValidateWindow(window []uint64) []uint64 {
	if len(window) > 20 {
		new_window := window[1:]
		return new_window
	}
	return window
}
