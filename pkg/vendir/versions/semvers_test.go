package versions_test

import (
	"reflect"
	"testing"

	ctlconf "github.com/k14s/vendir/pkg/vendir/config"
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
	}).Sorted().FilterConstraints(">0.0.5 <5.0.0")
	if err != nil {
		t.Fatalf("Expected filtering to succeed: %s", err)
	}

	expectedOrder := []string{
		"0.1.0",
		"2.0.0-10+meta.10", // prerelease is included
		"2.0.0-10",
		"2.0.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}

func TestSemverWithoutPrereleases(t *testing.T) {
	result := versions.NewSemvers([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).FilterPrereleases(nil)

	expectedOrder := []string{
		"0.1.0",
		"2.0.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}

func TestSemverWithPrereleases(t *testing.T) {
	preConf := &ctlconf.VersionSelectionSemverPrereleases{}

	result := versions.NewSemvers([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).FilterPrereleases(preConf)

	expectedOrder := []string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-pre.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}

func TestSemverWithPrereleaseIdentifiers(t *testing.T) {
	preConf := &ctlconf.VersionSelectionSemverPrereleases{Identifiers: []string{"alpha", "rc"}}

	result := versions.NewSemvers([]string{
		"2.0.0-10+meta.10",
		"0.0.1-pre.10",
		"0.0.1-alpha.1",
		"0.1.0",
		"2.0.0-10",
		"2.0.0",
		"0.0.1-rc.0",
	}).Sorted().FilterPrereleases(preConf)

	expectedOrder := []string{
		"0.0.1-alpha.1",
		"0.0.1-rc.0",
		"0.1.0",
		"2.0.0",
	}

	if !reflect.DeepEqual(result.All(), expectedOrder) {
		t.Fatalf("Expected result '%#v' to equal '%#v'", result.All(), expectedOrder)
	}
}
