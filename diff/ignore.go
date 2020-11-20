package diff

import (
	"encoding/json"
	"fmt"
	"github.com/aryann/difflib"
	"regexp"
	"sort"
)

type IgnoreManifest struct {
	ContentRegexp            string `json:"contentRegexp"`
	IgnoreSingleModification bool   `json:"singleModification"`
}

func (im *IgnoreManifest) String() string {
	return fmt.Sprintf("%+v", &im)
}

func (im *IgnoreManifest) Type() string {
	return "ignore"
}

func (im *IgnoreManifest) Set(value string) error {
	err := json.Unmarshal([]byte(value), im)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall ignore argument: %s", err)
	}
	return nil
}

type IgnoreManifests []struct {
	IdRegexp                 string `json:"idRegexp"`
	ContentRegexp            string `json:"contentRegexp"`
	IgnoreSingleModification bool   `json:"singleModification"`
}

func (im *IgnoreManifests) String() string {
	return fmt.Sprintf("%+v", &im)
}

func (im *IgnoreManifests) Type() string {
	return "ignoreMultipart"
}

func (im *IgnoreManifests) Set(value string) error {
	err := json.Unmarshal([]byte(value), im)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall ignoreMultipart argument: %s", err)
	}
	return nil
}

// TODO: consider pointer semantics for arguments
func getIgnoreParams(manifestId string, ima IgnoreManifest, ims IgnoreManifests) (contentRegexp string, ignoreSingleModification bool) {

	// Sort by precedence
	sort.Slice(ims, func(i, j int) bool {
		return len(ims[i].IdRegexp) > len(ims[j].IdRegexp)
	})

	for _, im := range ims {
		pattern := regexp.MustCompile(im.IdRegexp)
		if pattern.MatchString(manifestId) {
			return im.ContentRegexp, im.IgnoreSingleModification
		}
	}

	// If there is no ima, this one will be empty, which is desired
	return ima.ContentRegexp, ima.IgnoreSingleModification
}

func ignoreModificationsInternal(contentRegexp string, ignoreSingleModification bool, diffs []difflib.DiffRecord) (seenAnyChanges, ignoredAnyChanges bool) {
	if len(diffs) == 0 {
		return false, false
	} else if contentRegexp == "" {
		return true, false
	}

	seenAnyChanges = false
	for _, diff := range diffs {
		if diff.Delta != difflib.Common {
			seenAnyChanges = true
			break
		}
	}
	if !seenAnyChanges {
		return false, false
	}

	if ignoreSingleModification {
		pattern := regexp.MustCompile(contentRegexp)

		// Left only, right only count should match

		leftOnly := 0
		rightOnly := 0

		lastLeftIndex := 0
		for i, diff := range diffs {
			// need no zero based, for assertions in the next thing
			notZe := i + 1
			if diff.Delta == difflib.Common {
				continue
			}
			if diff.Delta != difflib.Common {
				if !pattern.MatchString(diff.Payload) {
					return true, false
				}
			}

			// only matched lines are here
			if diff.Delta == difflib.LeftOnly {
				leftOnly += 1
				lastLeftIndex = notZe
			} else if diff.Delta == difflib.RightOnly {
				if lastLeftIndex == 0 || lastLeftIndex != notZe-1 {
					return true, false
				}
				rightOnly += 1
			}
		}

		if leftOnly != rightOnly {
			return true, false
		}

		return false, true

	} else {
		pattern := regexp.MustCompile(contentRegexp)

		for _, diff := range diffs {
			if diff.Delta == difflib.LeftOnly {
				seenAnyChanges = true
				continue
			}

			if contentRegexp != "" && pattern.MatchString(diff.Payload) {
				ignoredAnyChanges = true
			}
		}

		if seenAnyChanges && ignoredAnyChanges {
			return false, true
		}

		return seenAnyChanges, ignoredAnyChanges
	}
}

func ignoreModifications(manifestId string, ima IgnoreManifest, ims IgnoreManifests, diffs []difflib.DiffRecord) (seenAnyChanges, ignoredAnyChanges bool) {
	contentRegexp, ignoreSingleModification := getIgnoreParams(manifestId, ima, ims)

	return ignoreModificationsInternal(contentRegexp, ignoreSingleModification, diffs)
}
