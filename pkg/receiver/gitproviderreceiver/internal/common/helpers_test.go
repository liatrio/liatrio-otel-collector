package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkSliceInt(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5, 6, 7, 8}

	expected := [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8}}

	chunks := ChunkSlice(slice, 3)

	assert.Equal(t, expected, chunks)
}

func TestChunkSliceFloat(t *testing.T) {
	slice := []float64{1.5, 2.5, 3.5, 4.5, 5.5, 6.5}

	expected := [][]float64{{1.5, 2.5, 3.5}, {4.5, 5.5, 6.5}}

	chunks := ChunkSlice(slice, 3)

	assert.Equal(t, expected, chunks)
}
func TestChunkSliceString(t *testing.T) {
	slice := []string{"a", "b", "c", "d", "e", "f"}

	expected := [][]string{{"a", "b", "c"}, {"d", "e", "f"}}

	chunks := ChunkSlice(slice, 3)

	assert.Equal(t, expected, chunks)
}

func TestChunkSliceSmall(t *testing.T) {
	slice := []int{1, 2}

	expected := [][]int{{1, 2}}

	chunks := ChunkSlice(slice, 3)

	assert.Equal(t, expected, chunks)
}
