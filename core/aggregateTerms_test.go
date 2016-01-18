package main

import (
	"testing"
)

func TestSingleTerm(t *testing.T) {
	expected := "snow"
	actual := aggregateTerms([]string{"snow"})

	if expected != actual {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}
}

func TestMultipleTerms(t *testing.T) {
	expected := "snow+ice+mountain"
	actual := aggregateTerms([]string{"snow", "ice", "mountain"})

	if expected != actual {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}
}
