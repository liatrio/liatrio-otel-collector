package githubscraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNumPages(t *testing.T) {
	n := float64(375)
	expected := 4

	num := getNumPages(n)

	assert.Equal(t, expected, num)

}
