package githubscraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNumPages100(t *testing.T) {
	p := float64(100)
	n := float64(375)

	expected := 4

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestGetNumPages10(t *testing.T) {
	p := float64(10)
	n := float64(375)

	expected := 38

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestGetNumPages1(t *testing.T) {
	p := float64(10)
	n := float64(1)

	expected := 1

	num := getNumPages(p, n)

	assert.Equal(t, expected, num)
}

func TestAdd100(t *testing.T) {
	a := 100
	b := 100

	expected := 200

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAdd10(t *testing.T) {
	a := 10
	b := 10

	expected := 20

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddNegative1(t *testing.T) {
	a := 1
	b := -1

	expected := 0

	num := add(a, b)

	assert.Equal(t, expected, num)
}

func TestAddNegative10(t *testing.T) {
	a := 1
	b := -10

	expected := -9

	num := add(a, b)

	assert.Equal(t, expected, num)
}
func TestGenDefaultSearchQueryOrg(t *testing.T) {
	st := "org"
	org := "empire"

	expected := "org:empire archived:false"

	actual := genDefaultSearchQuery(st, org)

	assert.Equal(t, expected, actual)
}

func TestGenDefaultSearchQueryUser(t *testing.T) {
	st := "user"
	org := "vader"

	expected := "user:vader archived:false"

	actual := genDefaultSearchQuery(st, org)

	assert.Equal(t, expected, actual)
}
