package cmd

import "errors"

// Error to report errors
type Error struct {
	error
	Code int
}

func handleDetailedExitCode(seenAnyChanges, ignoredAnyChanges bool) error {
	if seenAnyChanges {
		return Error{
			error: errors.New("identified at least one change, exiting with non-zero exit code (detailed-exitcode parameter enabled)"),
			Code:  2,
		}
	}

	if ignoredAnyChanges {
		return Error{
			error: errors.New("identified at least one ignored change, exiting with non-zero exit code (detailed-exitcode parameter enabled)"),
			Code:  3,
		}
	}

	return nil
}
