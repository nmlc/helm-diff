package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"k8s.io/helm/pkg/helm"

	"github.com/databus23/helm-diff/v3/diff"
	"github.com/databus23/helm-diff/v3/manifest"
)

type rollback struct {
	release          string
	client           helm.Interface
	detailedExitCode bool
	suppressedKinds  []string
	revisions        []string
	outputContext    int
	includeTests     bool
	showSecrets      bool
	ignore           diff.IgnoreManifest
	ignoreMultipart  diff.IgnoreManifests
	output           string
}

const rollbackCmdLongUsage = `
This command compares the laset manifests details of a named release
with specific revision values to rollback.

It forecasts/visualizes changes, that a helm rollback could perform.
`

func rollbackCmd() *cobra.Command {
	diff := rollback{}
	rollbackCmd := &cobra.Command{
		Use:     "rollback [flags] [RELEASE] [REVISION]",
		Short:   "Show a diff explaining what a helm rollback could perform",
		Long:    rollbackCmdLongUsage,
		Example: "  helm diff rollback my-release 2",
		PreRun: func(*cobra.Command, []string) {
			expandTLSPaths()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Suppress the command usage on error. See #77 for more info
			cmd.SilenceUsage = true

			if v, _ := cmd.Flags().GetBool("version"); v {
				fmt.Println(Version)
				return nil
			}

			if err := checkArgsLength(len(args), "release name", "revision number"); err != nil {
				return err
			}

			if q, _ := cmd.Flags().GetBool("suppress-secrets"); q {
				diff.suppressedKinds = append(diff.suppressedKinds, "Secret")
			}

			diff.release = args[0]
			diff.revisions = args[1:]

			if isHelm3() {
				return diff.backcastHelm3()
			}

			if diff.client == nil {
				diff.client = createHelmClient()
			}

			return diff.backcast()
		},
	}

	rollbackCmd.Flags().BoolP("suppress-secrets", "q", false, "suppress secrets in the output")
	rollbackCmd.Flags().BoolVar(&diff.showSecrets, "show-secrets", false, "do not redact secret values in the output")
	rollbackCmd.Flags().BoolVar(&diff.detailedExitCode, "detailed-exitcode", false, "return a non-zero exit code when there are changes")
	rollbackCmd.Flags().StringArrayVar(&diff.suppressedKinds, "suppress", []string{}, "allows suppression of the values listed in the diff output")
	rollbackCmd.Flags().IntVarP(&diff.outputContext, "context", "C", -1, "output NUM lines of context around changes")
	rollbackCmd.Flags().BoolVar(&diff.includeTests, "include-tests", false, "enable the diffing of the helm test hooks")
	rollbackCmd.Flags().StringVar(&diff.output, "output", "diff", "Possible values: diff, simple, template. When set to \"template\", use the env var HELM_DIFF_TPL to specify the template.")
	rollbackCmd.Flags().Var(&diff.ignore, "ignore", "TBD")
	rollbackCmd.Flags().Var(&diff.ignoreMultipart, "ignoreMultipart", "TBD")

	rollbackCmd.SuggestionsMinimumDistance = 1

	if !isHelm3() {
		addCommonCmdOptions(rollbackCmd.Flags())
	}

	return rollbackCmd
}

func (d *rollback) backcastHelm3() error {
	namespace := os.Getenv("HELM_NAMESPACE")
	excludes := []string{helm3TestHook, helm2TestSuccessHook}
	if d.includeTests {
		excludes = []string{}
	}
	// get manifest of the latest release
	releaseResponse, err := getRelease(d.release, namespace)

	if err != nil {
		return err
	}

	// get manifest of the release to rollback
	revision, _ := strconv.Atoi(d.revisions[0])
	revisionResponse, err := getRevision(d.release, revision, namespace)
	if err != nil {
		return err
	}

	// create a diff between the current manifest and the version of the manifest that a user is intended to rollback
	seenAnyChanges, ignoredAnyChanges := diff.Manifests(
		manifest.Parse(string(releaseResponse), namespace, excludes...),
		manifest.Parse(string(revisionResponse), namespace, excludes...),
		d.suppressedKinds,
		d.showSecrets,
		d.outputContext,
		d.ignore,
		d.ignoreMultipart,
		d.output,
		os.Stdout,
	)

	if d.detailedExitCode {
		return handleDetailedExitCode(seenAnyChanges, ignoredAnyChanges)
	}
	return nil
}

func (d *rollback) backcast() error {

	// get manifest of the latest release
	releaseResponse, err := d.client.ReleaseContent(d.release)

	if err != nil {
		return prettyError(err)
	}

	// get manifest of the release to rollback
	revision, _ := strconv.Atoi(d.revisions[0])
	revisionResponse, err := d.client.ReleaseContent(d.release, helm.ContentReleaseVersion(int32(revision)))
	if err != nil {
		return prettyError(err)
	}

	// create a diff between the current manifest and the version of the manifest that a user is intended to rollback
	seenAnyChanges, ignoredAnyChanges := diff.Manifests(
		manifest.ParseRelease(releaseResponse.Release, d.includeTests),
		manifest.ParseRelease(revisionResponse.Release, d.includeTests),
		d.suppressedKinds,
		d.showSecrets,
		d.outputContext,
		d.ignore,
		d.ignoreMultipart,
		d.output,
		os.Stdout,
	)

	if d.detailedExitCode {
		return handleDetailedExitCode(seenAnyChanges, ignoredAnyChanges)
	}
	return nil
}
