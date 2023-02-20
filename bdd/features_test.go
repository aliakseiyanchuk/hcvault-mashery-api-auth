package bdd_test

import (
	"github.com/cucumber/godog"
	"github.com/rdumont/assistdog"
	"strings"
	"testing"
)

var assist *assistdog.Assist

func init() {
	assist = assistdog.NewDefault()

	var b bool
	var b1 *bool

	assist.RegisterParser(b, func(raw string) (interface{}, error) {
		switch strings.ToLower(raw) {
		case "true", "\"true\"", "yes", "\"yes\"":
			return true, nil
		default:
			return false, nil
		}
	})

	assist.RegisterParser(b1, func(raw string) (interface{}, error) {
		rv := false
		switch strings.ToLower(raw) {
		case "true", "\"true\"", "yes", "\"yes\"":
			rv = true
		default:
			// Nothing to do
		}

		return &rv, nil
	})
}

func TestOAEPFeatures(t *testing.T) {
	scenarioMountPoint := "mash-auth-oaep"
	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			// Add step definitions here.
			setupScenarioSteps(s)

			mountSecretEngineForScenario(s, scenarioMountPoint)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/oaep"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func TestRoleCRUDFeatures(t *testing.T) {
	scenarioMountPoint := "mash-auth-crud"
	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			// Add step definitions here.
			setupScenarioSteps(s)

			mountSecretEngineForScenario(s, scenarioMountPoint)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/role_crud"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func TestConfigFeatures(t *testing.T) {
	scenarioMountPoint := "mash-auth-cfg"
	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			// Add step definitions here.
			setupScenarioSteps(s)

			mountSecretEngineForScenario(s, scenarioMountPoint)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/config/config.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func TestRateControlFeatures(t *testing.T) {
	scenarioMountPoint := "mash-auth-rate-ctrl"
	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			// Add step definitions here.
			setupScenarioSteps(s)

			mountSecretEngineForScenario(s, scenarioMountPoint)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/rate_ctrl/proxy-mode-control.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func TestImportExportFeatures(t *testing.T) {
	scenarioMountPoint := "mash-auth-impexp"
	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			// Add step definitions here.
			setupScenarioSteps(s)

			mountSecretEngineForScenario(s, scenarioMountPoint)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/impexp"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
