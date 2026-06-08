package repository

import (
	"strconv"
	"strings"
)

// EncodeVector converts a float32 slice to pgvector text format.
func EncodeVector(vector []float32) string {
	parts := make([]string, len(vector))
	for i, value := range vector {
		parts[i] = strconv.FormatFloat(float64(value), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
