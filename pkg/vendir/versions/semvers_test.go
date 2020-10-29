package versions_test

import (
	"testing"
	"reflect"

	"github.com/k14s/vendir/pkg/vendir/versions"
)

func TestSemverOrder(t *testing.T) {
	result := versions.NewSemvers([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"v2.0.0", // prefixed with v
		"0.0.1-rc.0",
	}).Sorted().All()

	expectedOrder := []string{
		"0.0.1-pre.1",
		"0.0.1-pre.10",
		"0.0.1-rc.0",
		"0.1.0",
		"2.0.0-10+meta.10",
		"2.0.0-10",
		"2.0.0",
		"v2.0.0",
	}

	if !reflect.DeepEqual(result, expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result, expectedOrder)
	}
}

func TestSemverFilter(t *testing.T) {
	result, err := versions.NewSemvers([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).Filtered(">0.0.5, <5.0.0")
	if err != nil {
		t.Fatalf("Expected filtering to succeed")
	}

	expectedOrder := []string{
		"0.1.0",
		// TODO prereleases arent included
		// "2.0.0-10+meta.10",
		// "2.0.0-10",
		"2.0.0",
		// "v2.0.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}
