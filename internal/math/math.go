package mymath

import (
	"slices"
	"time"
)

type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

func Clamp[T number](minV, V, maxV T) T {
	return max(min(V, maxV), minV)
}

func Median[T number](values []T) T {
	slices.Sort(values)
	switch len(values) {
	case 0:
		return 0
	case 1, 2:
		return values[0]
	default:
		i := Clamp(0, len(values)/2-1, len(values)-1)
		return values[i]
	}
}

func MedianTime(values []time.Time) time.Time {
	slices.SortFunc(values, func(a, b time.Time) int {
		return int(int64(a.Sub(b)) / int64(time.Second))
	})
	switch len(values) {
	case 0:
		return time.Time{}
	case 1, 2:
		return values[0]
	default:
		i := Clamp(0, len(values)/2-1, len(values)-1)
		return values[i]
	}
}
