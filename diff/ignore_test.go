package diff

import (
	"github.com/mgutz/ansi"
	"testing"
)

func assertManifestsReturnValues(t *testing.T, changesSeen, changesSeenDesired, changesIgnored, changesIgnoredDesired bool) {
	t.Helper()

	if changesSeen != changesSeenDesired {
		t.Errorf("Unexpected return value from Manifests: Expected `changesSeen` to be `%t`, but was `%t`", changesSeenDesired, changesSeen)
	}

	if changesIgnored != changesIgnoredDesired {
		t.Errorf("Unexpected return value from Manifests: Expected `changesIgnored` to be `%t`, but was `%t`", changesIgnoredDesired, changesIgnored)
	}
}

func TestIgnoreManifest(t *testing.T) {
	ansi.DisableColors(true)

	specBeta := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: nginx
`
	specDeleteRegexMatch := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
`
	specModifySingleLine := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: nginx - modify
`

	cases := []struct {
		description    string
		newIndex       string
		ignoreRegex    string
		changesSeen    bool
		changesIgnored bool
	}{
		{"Regex match deleted", specDeleteRegexMatch, "nginx", true, false},
		{"Regex match on changes", specModifySingleLine, "nginx", false, true},
		{"Regex match on no changes", specModifySingleLine, "kind", false, true},
		{"Regex irrelevant", specModifySingleLine, "irrelevant", true, false},
		{"No changes", specBeta, "nginx", false, false},
		// TODO: add test if ignoreRegex is not specified, or this should be tested by other test cases? yeah, I think so, or this should be tested by other test cases? yeah, I think so.
	}

	for _, test := range cases {
		t.Run(test.description, func(t *testing.T) {
			diffs := diffStrings(specBeta, test.newIndex)
			changesSeen, changesIgnored := ignoreModificationsInternal(test.ignoreRegex, false, diffs)
			assertManifestsReturnValues(t, changesSeen, test.changesSeen, changesIgnored, test.changesIgnored)
		})
	}
}

func TestIgnoreSingleLineModificationFunction(t *testing.T) {
	ansi.DisableColors(true)

	specBeta := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: nginx
test:
  name: nginx
`
	specModifyMatchLine := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: nginx - modify
test:
  name: nginx
`
	specModifyMultipleMatchLine := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: nginx - modify
test:
  name: nginx - modify
`
	specModifyNotMatchLine :=
		`
apiVersion: apps/v1beta1
kind: Deployment - modify
metadata:
  name: nginx
test:
  name: nginx
`
	specModifyNotMatchLineAndMatchLine := `
apiVersion: apps/v1beta1
kind: Deployment - modify
metadata:
  name: nginx - modify
test:
  name: nginx
`
	specMovingMatchLine := `
apiVersion: apps/v1beta1
  name: nginx
kind: Deployment
metadata:
test:
  name: nginx
`
	specDeleteMatchLine := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
test:
  name: nginx
`
	specAddMatchLine := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: nginx
  name: nginx
test:
  name: nginx
`

	cases := []struct {
		description    string
		newIndex       string
		ignoreRegex    string
		changesSeen    bool
		changesIgnored bool
	}{
		{"No changes", specBeta, "nginx", false, false},
		{"Modify match line", specModifyMatchLine, "nginx", false, true},
		{"Move match line, should be subsequent lines", specMovingMatchLine, "nginx", true, false},
		{"Delete match line", specDeleteMatchLine, "nginx", true, false},
		{"Add match line", specAddMatchLine, "nginx", true, false},
		{"Modify multiple mattch line", specModifyMultipleMatchLine, "nginx", false, true},
		{"Modify not match line", specModifyNotMatchLine, "nginx", true, false},
		{"Modify not match line and match line", specModifyNotMatchLineAndMatchLine, "nginx", true, false},
		// TODO: add test if ignoreRegex is not specified, or this should be tested by other test cases? yeah, I think so, or this should be tested by other test cases? yeah, I think so.
	}

	for _, test := range cases {
		t.Run(test.description, func(t *testing.T) {
			diffs := diffStrings(specBeta, test.newIndex)
			changesSeen, changesIgnored := ignoreModificationsInternal(test.ignoreRegex, true, diffs)
			assertManifestsReturnValues(t, changesSeen, test.changesSeen, changesIgnored, test.changesIgnored)
		})
	}
}

func TestIgnoreParams(t *testing.T) {
	ansi.DisableColors(true)

	cases := []struct {
		description              string
		im                       IgnoreManifest
		ims                      IgnoreManifests
		contentRegexp            string
		ignoreSingleModification bool
	}{
		{"`all` works", IgnoreManifest{"nginx", false}, IgnoreManifests{{"Deploymentx", "nginx", true}}, "nginx", false},
		{"`all` have lower precedence", IgnoreManifest{"nginx", false}, IgnoreManifests{{"Deployment", "nginx", true}}, "nginx", true},
		{"precedence by length", IgnoreManifest{}, IgnoreManifests{{"Deployment", "nginx", true}, {"nginx, Deployment", "nginxy", false}}, "nginxy", false},
	}

	for _, test := range cases {
		t.Run(test.description, func(t *testing.T) {
			contentRegexp, ignoreSingleModificaiton := getIgnoreParams("default, nginx, Deployment (apps)", test.im, test.ims)

			if contentRegexp != test.contentRegexp {
				t.Errorf("Unexpected return value from Manifests: Expected `changesSeen` to be")
			}

			if ignoreSingleModificaiton != test.ignoreSingleModification {
				t.Errorf("Unexpected return value from Manifests: Expected `changesIgnored` to be")
			}
		})
	}
}
