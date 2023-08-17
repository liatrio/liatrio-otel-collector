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

func TestGenDefaultSearchQueryOrg(t *testing.T) {
	st := "org"
	org := "empire"

	expected := "org:empire"

	actual := genDefaultSearchQuery(st, org)

	assert.Equal(t, expected, actual)
}

func TestGenDefaultSearchQueryUser(t *testing.T) {
	st := "user"
	org := "vader"

	expected := "user:vader"

	actual := genDefaultSearchQuery(st, org)

	assert.Equal(t, expected, actual)
}
