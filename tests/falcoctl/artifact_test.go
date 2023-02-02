package testfalcoctl

import (
	"strings"
	"testing"

	"github.com/jasondellaluce/falco-testing/pkg/falco"
	"github.com/jasondellaluce/falco-testing/pkg/falcoctl"
	"github.com/jasondellaluce/falco-testing/pkg/run"
	"github.com/jasondellaluce/falco-testing/tests"
	"github.com/jasondellaluce/falco-testing/tests/data/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifact_InstallPlugin(t *testing.T) {
	t.Parallel()

	t.Run("fail-missing-arg", func(t *testing.T) {
		t.Parallel()
		res := falcoctl.Test(
			tests.NewFalcoctlExecutableRunner(t),
			falcoctl.WithArgs("artifact", "install"),
			falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
		)
		assert.NotNil(t, res.Err(), "%s", res.Stdout())
		assert.NotZero(t, res.ExitCode())
		assert.Contains(t, res.Stdout(), "no artifacts to install")
	})

	t.Run("fail-invalid-artifact", func(t *testing.T) {
		t.Parallel()
		res := falcoctl.Test(
			tests.NewFalcoctlExecutableRunner(t),
			falcoctl.WithArgs("artifact", "install", "some_invalid_artifact"),
			falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
		)
		assert.NotNil(t, res.Err(), "%s", res.Stdout())
		assert.NotZero(t, res.ExitCode())
		assert.Contains(t, res.Stdout(), "cannot find some_invalid_artifact")
	})

	t.Run("install-plugin", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "install", "dummy"),
				falcoctl.WithPluginsDir(sharedWorkDir+"/plugins"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.Contains(t, res.Stdout(), "Extracting and installing \"plugin\" \"dummy")
			assert.Contains(t, res.Stdout(), "Artifact successfully installed")
			assert.FileExists(t, sharedWorkDir+"/plugins/libdummy.so")
		}))
	})

	t.Run("install-rules-with-deps", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "install", "cloudtrail-rules"),
				falcoctl.WithPluginsDir(sharedWorkDir+"/plugins"),
				falcoctl.WithRulesFilesDir(sharedWorkDir+"/rulesfiles"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.Contains(t, res.Stdout(), "Extracting and installing \"plugin\" \"cloudtrail")
			assert.Contains(t, res.Stdout(), "Extracting and installing \"plugin\" \"json")
			assert.Contains(t, res.Stdout(), "Artifact successfully installed")
			assert.FileExists(t, sharedWorkDir+"/rulesfiles/aws_cloudtrail_rules.yaml")
			assert.FileExists(t, sharedWorkDir+"/plugins/libcloudtrail.so")
			assert.FileExists(t, sharedWorkDir+"/plugins/libjson.so")
		}))
	})

	t.Run("install-for-falco", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "install", "cloudtrail-rules"),
				falcoctl.WithPluginsDir(sharedWorkDir+"/plugins"),
				falcoctl.WithRulesFilesDir(sharedWorkDir+"/rulesfiles"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			require.Nil(t, res.Err(), "%s", res.Stdout())
			require.Zero(t, res.ExitCode())

			// craft a configuration for the plugin
			config, err := falco.NewPluginConfig(
				&falco.PluginConfigInfo{
					Name:    "cloudtrail",
					Library: plugins.CloudtrailPlugin.Name(),
				},
				&falco.PluginConfigInfo{
					Name:    "json",
					Library: plugins.JSONPlugin.Name(),
				},
			)
			require.Nil(t, err)

			// launch Falco with the installed plugin and validate its ruleset
			jsonFile := run.NewLocalFileAccessor("libjson.so", sharedWorkDir+"/plugins/libjson.so")
			cloudtrailFile := run.NewLocalFileAccessor("libcloudtrail.so", sharedWorkDir+"/plugins/libcloudtrail.so")
			rulesFile := run.NewLocalFileAccessor("aws_rules.yaml", sharedWorkDir+"/rulesfiles/aws_cloudtrail_rules.yaml")
			resFalco := falco.Test(
				tests.NewFalcoExecutableRunner(t),
				falco.WithOutputJSON(),
				falco.WithConfig(config),
				falco.WithExtraFiles(jsonFile, cloudtrailFile),
				falco.WithRulesValidation(rulesFile),
				falco.WithEnabledSources("aws_cloudtrail"),
			)
			assert.Nil(t, resFalco.Err(), "%s", resFalco.Stderr())
			assert.Equal(t, 0, resFalco.ExitCode())
			assert.True(t, resFalco.RuleValidation().ForIndex(0).Successful)
			assert.Zero(t, resFalco.RuleValidation().AllWarnings().Count())
		}))
	})
}

func TestArtifact_Info(t *testing.T) {
	t.Parallel()

	t.Run("fail-missing-arg", func(t *testing.T) {
		t.Parallel()
		res := falcoctl.Test(
			tests.NewFalcoctlExecutableRunner(t),
			falcoctl.WithArgs("artifact", "info"),
			falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
		)
		assert.NotNil(t, res.Err(), "%s", res.Stderr())
		assert.NotZero(t, res.ExitCode())
		assert.Contains(t, res.Stderr(), "Error: requires at least 1 arg")
	})

	t.Run("info-plugin", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "info", "dummy"),
				falcoctl.WithPluginsDir(sharedWorkDir+"/plugins"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.Regexp(t, `.*REF[\s]+TAGS.*`, res.Stdout())
			assert.Regexp(t, `.*ghcr.io\/falcosecurity\/plugins\/plugin\/dummy[\s]*([0-9]+.[0-9]+.[0-9]+[\s]*)+latest.*`, res.Stdout())
			assert.NoFileExists(t, sharedWorkDir+"/plugins/libdummy.so")
		}))
	})

	t.Run("info-rules", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "info", "cloudtrail-rules"),
				falcoctl.WithPluginsDir(sharedWorkDir+"/plugins"),
				falcoctl.WithRulesFilesDir(sharedWorkDir+"/rulesfiles"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.Regexp(t, `.*REF[\s]+TAGS.*`, res.Stdout())
			assert.Regexp(t, `.*ghcr.io\/falcosecurity\/plugins\/ruleset\/cloudtrail[\s]*([0-9]+.[0-9]+.[0-9]+[\s]*)+latest.*`, res.Stdout())
			assert.NoFileExists(t, sharedWorkDir+"/plugins/libcloudtrail.so")
			assert.NoFileExists(t, sharedWorkDir+"/rulesfiles/aws_cloudtrail_rules.yaml")
		}))
	})
}

func TestArtifact_List(t *testing.T) {
	t.Parallel()

	t.Run("list-all", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "list"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.GreaterOrEqual(t, len(strings.Split(res.Stdout(), "\n")), 2)
			assert.Regexp(t, `.*INDEX[\s]+ARTIFACT[\s]+TYPE[\s]+REGISTRY[\s]+REPOSITORY.*`, res.Stdout())
			assert.Regexp(t, `.*falcosecurity[\s]+dummy[\s]+plugin[\s]+ghcr.io[\s]+falcosecurity/plugins/plugin/dummy.*`, res.Stdout())
		}))
	})

	t.Run("list-plugins", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "list", "--type=plugin"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.GreaterOrEqual(t, len(strings.Split(res.Stdout(), "\n")), 2)
			assert.Regexp(t, `.*INDEX[\s]+ARTIFACT[\s]+TYPE[\s]+REGISTRY[\s]+REPOSITORY.*`, res.Stdout())
			assert.Regexp(t, `.*falcosecurity[\s]+dummy[\s]+plugin[\s]+ghcr.io[\s]+falcosecurity/plugins/plugin/dummy.*`, res.Stdout())
			assert.NotRegexp(t, `.*falcosecurity.*rulesfile[\s]+ghcr.io.*`, res.Stdout())
		}))
	})

	t.Run("list-rules", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "list", "--type=rulesfile"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.GreaterOrEqual(t, len(strings.Split(res.Stdout(), "\n")), 2)
			assert.Regexp(t, `.*INDEX[\s]+ARTIFACT[\s]+TYPE[\s]+REGISTRY[\s]+REPOSITORY.*`, res.Stdout())
			assert.Regexp(t, `.*falcosecurity[\s]+cloudtrail-rules[\s]+rulesfile[\s]+ghcr.io[\s]+falcosecurity/plugins/ruleset/cloudtrail.*`, res.Stdout())
			assert.NotRegexp(t, `.*falcosecurity.*plugin[\s]+ghcr.io.*`, res.Stdout())
		}))
	})
}

func TestArtifact_Search(t *testing.T) {
	t.Parallel()

	t.Run("fail-missing-arg", func(t *testing.T) {
		t.Parallel()
		res := falcoctl.Test(
			tests.NewFalcoctlExecutableRunner(t),
			falcoctl.WithArgs("artifact", "search"),
			falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
		)
		assert.NotNil(t, res.Err(), "%s", res.Stderr())
		assert.NotZero(t, res.ExitCode())
		assert.Contains(t, res.Stderr(), "Error: requires at least 1 arg")
	})

	t.Run("seach-dummy", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "search", "dummy"),
				falcoctl.WithPluginsDir(sharedWorkDir+"/plugins"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.Regexp(t, `.*INDEX[\s]+ARTIFACT[\s]+TYPE[\s]+REGISTRY[\s]+REPOSITORY.*`, res.Stdout())
			assert.Regexp(t, `.*falcosecurity[\s]+dummy[\s]+plugin[\s]+ghcr.io[\s]+falcosecurity/plugins/plugin/dummy.*`, res.Stdout())
			assert.Regexp(t, `.*falcosecurity[\s]+dummy_c[\s]+plugin[\s]+ghcr.io[\s]+falcosecurity/plugins/plugin/dummy_c.*`, res.Stdout())
			assert.NoFileExists(t, sharedWorkDir+"/plugins/libdummy.so")
			assert.NoFileExists(t, sharedWorkDir+"/plugins/libdummy_c.so")
		}))
	})

	t.Run("seach-dummy-maximum-score", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			res := falcoctl.Test(
				tests.NewFalcoctlExecutableRunner(t),
				falcoctl.WithArgs("artifact", "search", "dummy", "--min-score=1"),
				falcoctl.WithPluginsDir(sharedWorkDir+"/plugins"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, res.Err(), "%s", res.Stdout())
			assert.Zero(t, res.ExitCode())
			assert.Regexp(t, `.*INDEX[\s]+ARTIFACT[\s]+TYPE[\s]+REGISTRY[\s]+REPOSITORY.*`, res.Stdout())
			assert.Regexp(t, `.*falcosecurity[\s]+dummy[\s]+plugin[\s]+ghcr.io[\s]+falcosecurity/plugins/plugin/dummy.*`, res.Stdout())
			assert.NotRegexp(t, `.*falcosecurity[\s]+dummy_c[\s]+plugin[\s]+ghcr.io[\s]+falcosecurity/plugins/plugin/dummy_c.*`, res.Stdout())
			assert.NoFileExists(t, sharedWorkDir+"/plugins/libdummy.so")
		}))
	})

	t.Run("search-all", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, run.WorkDir(func(sharedWorkDir string) {
			runner := tests.NewFalcoctlExecutableRunner(t)
			resList := falcoctl.Test(
				runner,
				falcoctl.WithArgs("artifact", "list"),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			resSearch := falcoctl.Test(
				runner,
				falcoctl.WithArgs("artifact", "seatch", ""),
				falcoctl.WithConfig(run.NewStringFileAccessor("config.yaml", "")),
			)
			assert.Nil(t, resList.Err(), "%s", resList.Stdout())
			assert.Nil(t, resSearch.Err(), "%s", resSearch.Stdout())
			assert.Zero(t, resList.ExitCode())
			assert.Zero(t, resSearch.ExitCode())
			listLines := strings.Split(resSearch.Stdout(), "\n")
			searchLines := strings.Split(resSearch.Stdout(), "\n")
			for _, line := range listLines {
				found := false
				for _, searchline := range searchLines {
					if line == searchline {
						found = true
					}
				}
				assert.True(t, found, "empty search must have same output as list")
			}
		}))
	})
}
