package runner

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"encoding/json"

	"github.com/pkg/errors"
)

const (
	CombinedShScript         = "./fixture/combined.sh"
	CombinedPowershellScript = "./fixture/combined.ps1"
)

func TestConfigLoadConfig(t *testing.T) {
	c := &Runner{}
	if err := c.LoadFromFile("./fixture/checks.json"); err != nil {
		t.Fatal(err)
	}
}

func TestNewRunner(t *testing.T) {
	// Assert that the only allowed roles are master and agent.
	var (
		r   *Runner
		err error
	)

	// NewRunner() should succeed with these roles.
	for _, role := range []string{"master", "agent"} {
		r, err = NewRunner(role)
		if err != nil {
			t.Fatal(err)
		}

		if r.role != role {
			t.Fatalf("Expected runner role %s. Got %s", role, r.role)
		}
	}

	// NewRunner() should return an error with these roles.
	for _, role := range []string{"", "foo", "agent_public"} {
		r, err = NewRunner(role)
		if err == nil {
			t.Fatalf("NewRunner(\"%s\") should return an error but does not", role)
		}
	}
}

// keys returns an array of m's keys.
func keys(m map[string]interface{}) []string {
	s := make([]string, len(m))
	i := 0
	for k := range m {
		s[i] = k
		i++
	}
	return s
}

func TestRun(t *testing.T) {
	checkScript := CombinedShScript
	expectedCheckOutput := "STDOUT\nSTDERR\n"
	if runtime.GOOS == "windows" {
		checkScript = CombinedPowershellScript
		expectedCheckOutput = "STDOUT\r\nSTDERR\r\n"
	}

	// Build the check config for this test.
	checkCfg := map[string]map[string]interface{}{
		"cluster_checks": map[string]interface{}{
			"check1": map[string]interface{}{
				"cmd":     []string{checkScript},
				"timeout": "1s",
			},
			"check2": map[string]interface{}{
				"cmd":     []string{checkScript},
				"timeout": "1s",
			},
			"check3": map[string]interface{}{
				"cmd":     []string{checkScript},
				"timeout": "1s",
			},
			"check4": map[string]interface{}{
				"cmd":     []string{checkScript},
				"timeout": "1s",
			},
		},
		"node_checks": map[string]interface{}{
			"checks": map[string]interface{}{
				"check5": map[string]interface{}{
					"cmd":     []string{checkScript},
					"timeout": "1s",
				},
				"check6": map[string]interface{}{
					"cmd":     []string{checkScript},
					"timeout": "1s",
					"roles":   []string{"master", "agent"},
				},
				"check7": map[string]interface{}{
					"cmd":     []string{checkScript},
					"timeout": "1s",
					"roles":   []string{"master"},
				},
				"check8": map[string]interface{}{
					"cmd":     []string{checkScript},
					"timeout": "1s",
					"roles":   []string{"agent"},
				},
			},
			"prestart":  []string{"check5", "check6", "check7", "check8"},
			"poststart": []string{"check5", "check6", "check7", "check8"},
		},
	}
	checkCfgJSON, err := json.Marshal(checkCfg)
	if err != nil {
		t.Fatal(err)
	}

	// From the config, build lists of check names we expect by type and role.
	// Cluster checks aren't filtered by role, so we expect all cluster checks on all roles.
	expectedClusterChecks := keys(checkCfg["cluster_checks"])
	// Node checks may be filtered by role.
	// To simplify the test, we assume that all node checks are both prestart and poststart.
	expectedNodeChecks := map[string][]string{
		"master": []string{},
		"agent":  []string{},
	}
	nodeChecksVal, ok := checkCfg["node_checks"]["checks"]
	if !ok {
		t.Fatalf("Node checks not present in config")
	}
	nodeChecks := nodeChecksVal.(map[string]interface{})
	for checkName, checkDefVal := range nodeChecks {
		checkDef := checkDefVal.(map[string]interface{})
		rolesVal, ok := checkDef["roles"]
		if !ok {
			// No role specified, so we expect it on all roles.
			for role, checkNames := range expectedNodeChecks {
				expectedNodeChecks[role] = append(checkNames, checkName)
			}
		} else {
			roles := rolesVal.([]string)
			for _, role := range roles {
				checkNames, ok := expectedNodeChecks[role]
				if !ok {
					t.Fatalf("Unexpected role %s for node check %s", role, checkName)
				}
				expectedNodeChecks[role] = append(checkNames, checkName)
			}
		}
	}

	// For each role, instantiate a check runner, run all types of checks, and then verify the results.
	for _, role := range []string{"master", "agent"} {
		r, err := NewRunner(role)
		if err != nil {
			t.Fatal(err)
		}
		err = r.Load(strings.NewReader(string(checkCfgJSON[:])))
		if err != nil {
			t.Fatal(err)
		}

		// Cluster checks
		clusterCheckResponse, err := r.Cluster(context.TODO(), false)
		if err != nil {
			t.Fatal(err)
		}
		if clusterCheckResponse.Status() != 0 {
			t.Fatalf("Expected status 0 for %s cluster checks, got %d", role, clusterCheckResponse.Status())
		}
		if len(clusterCheckResponse.checks) != len(expectedClusterChecks) {
			t.Fatalf("Expected %d checks in %s cluster checks response, got %d", len(expectedClusterChecks), role, len(clusterCheckResponse.checks))
		}
		for _, checkName := range expectedClusterChecks {
			if err := validateCheck(checkName, 0, expectedCheckOutput, clusterCheckResponse.checks); err != nil {
				t.Fatal(err)
			}
		}

		// Prestart node checks
		prestartNodeCheckResponse, err := r.PreStart(context.TODO(), false)
		if err != nil {
			t.Fatal(err)
		}
		if prestartNodeCheckResponse.Status() != 0 {
			t.Fatalf("Expected status 0 for %s node-prestart checks, got %d", role, prestartNodeCheckResponse.Status())
		}
		if len(prestartNodeCheckResponse.checks) != len(expectedNodeChecks[role]) {
			t.Fatalf("Expected %d checks in %s node-prestart checks response, got %d", len(expectedNodeChecks[role]), role, len(prestartNodeCheckResponse.checks))
		}
		for _, checkName := range expectedNodeChecks[role] {
			if err := validateCheck(checkName, 0, expectedCheckOutput, prestartNodeCheckResponse.checks); err != nil {
				t.Fatal(err)
			}
		}

		// Poststart node checks
		poststartNodeCheckResponse, err := r.PostStart(context.TODO(), false)
		if err != nil {
			t.Fatal(err)
		}
		if poststartNodeCheckResponse.Status() != 0 {
			t.Fatalf("Expected status 0 for %s node-poststart checks, got %d", role, poststartNodeCheckResponse.Status())
		}
		if len(poststartNodeCheckResponse.checks) != len(expectedNodeChecks[role]) {
			t.Fatalf("Expected %d checks in %s node-poststart checks response, got %d", len(expectedNodeChecks[role]), role, len(poststartNodeCheckResponse.checks))
		}
		for _, checkName := range expectedNodeChecks[role] {
			if err := validateCheck(checkName, 0, expectedCheckOutput, poststartNodeCheckResponse.checks); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// validateCheck takes the name of a check, its expected status and output, as well as a map of check results, and verifies the check is included in the results with the expected status and output.
func validateCheck(checkName string, expectedStatus int, expectedOutput string, checkResults map[string]*Response) error {
	checkResult, ok := checkResults[checkName]
	if !ok {
		return errors.Errorf("Response is missing check %s", checkName)
	}
	if checkResult.status != 0 {
		return errors.Errorf("Expected status 0 for check %s, got %d", checkName, checkResult.status)
	}
	if checkResult.output != expectedOutput {
		return errors.Errorf("Expected output \"%s\" for check %s, got \"%s\"", expectedOutput, checkName, checkResult.output)
	}
	return nil
}

func TestList(t *testing.T) {
	r, err := NewRunner("master")
	if err != nil {
		t.Fatal(err)
	}

	cfg := `
{
  "cluster_checks": {
    "cluster_check_1": {
      "description": "Cluster check 1",
      "cmd": ["echo", "cluster_check_1"],
      "timeout": "1s"
    }
  },
  "node_checks": {
    "checks": {
      "node_check_1": {
        "description": "Node check 1",
        "cmd": ["echo", "node_check_1"],
        "timeout": "1s"
      },
      "node_check_2": {
        "description": "Node check 2",
        "cmd": ["echo", "node_check_2"],
        "timeout": "1s",
        "roles": ["master"]
      },
      "node_check_3": {
        "description": "Node check 3",
        "cmd": ["echo", "node_check_3"],
        "timeout": "1s",
        "roles": ["agent"]
      }
    },
    "prestart": ["node_check_1"],
    "poststart": ["node_check_2", "node_check_3"]
  }
}`
	r.Load(strings.NewReader(cfg))

	out, err := r.Cluster(context.TODO(), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCheckListing(out, "cluster_check_1", "Cluster check 1", "1s", []string{"echo", "cluster_check_1"}); err != nil {
		t.Fatal(err)
	}

	out, err = r.PreStart(context.TODO(), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCheckListing(out, "node_check_1", "Node check 1", "1s", []string{"echo", "node_check_1"}); err != nil {
		t.Fatal(err)
	}

	out, err = r.PostStart(context.TODO(), true)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCheckListing(out, "node_check_2", "Node check 2", "1s", []string{"echo", "node_check_2"}); err != nil {
		t.Fatal(err)
	}

	// This runner is for a master, so a check that only runs on agents should not be listed.
	unexpectedCheckName := "node_check_3"
	if _, ok := out.checks[unexpectedCheckName]; ok {
		t.Fatalf("found unexpected check %s", unexpectedCheckName)
	}
}

func validateCheckListing(cr *CombinedResponse, name, description, timeout string, cmd []string) error {
	check, ok := cr.checks[name]
	if !ok {
		return errors.Errorf("expect check %s", name)
	}

	if check.description != description {
		return errors.Errorf("expect description %s. Got %s", description, check.description)
	}

	if check.timeout != timeout {
		return errors.Errorf("expect timeout %s. Got %s", timeout, check.timeout)
	}

	for i := range check.cmd {
		if check.cmd[i] != cmd[i] {
			return errors.Errorf("expect cmd %s. Got %s", cmd, check.cmd)
		}
	}

	return nil
}

func TestTimeout(t *testing.T) {
	r, err := NewRunner("master")
	if err != nil {
		t.Fatal(err)
	}

	cfg := `
{
  "node_checks": {
    "checks": {
      "check1": {
        "cmd": ["./fixture/combined.sh"],
        "timeout": "1s"
      },
      "check2": {
        "cmd": ["./fixture/inf2.sh"],
        "timeout": "500ms"
      }
    },
    "poststart": ["check1", "check2"]
  }
}`

	if runtime.GOOS == "windows" {
		t.Skip("TestTimeout was skipped on Windows")
	}
	err = r.Load(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}

	out, err := r.PostStart(context.TODO(), false)
	if err != nil {
		t.Fatal(err)
	}

	// marshal the check output
	mOut, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}

	type expectedOutput struct {
		Status int `json:"status"`
		Checks map[string]struct {
			Output string `json:"output"`
			Status int    `json:"status"`
		} `json:"checks"`
	}

	var resp expectedOutput

	if err := json.Unmarshal(mOut, &resp); err != nil {
		t.Fatal(err)
	}

	expectedErrMsg := "command [./fixture/inf2.sh] exceeded timeout 500ms and was killed"
	check2, ok := resp.Checks["check2"]
	if !ok {
		t.Fatal("check2 not found in response")
	}

	if check2.Status != statusUnknown {
		t.Fatalf("expect check2 status %d. Got %d", statusUnknown, check2.Status)
	}
	if check2.Output != expectedErrMsg {
		t.Fatalf("expect output %s. Got %s", expectedErrMsg, check2.Output)
	}
}

// runWithTimeout calls f() and returns an error if it takes longer than d to return.
func runWithTimeout(d time.Duration, f func()) error {
	finished := make(chan bool, 1)
	go func() {
		f()
		finished <- true
	}()
	select {
	case <-finished:
		return nil
	case <-time.After(d):
		return errors.New(fmt.Sprintf("timed out after %s", d.String()))
	}
}

// TestRunnerParallelism verifies that the check runner runs checks in parallel, using timeouts.
func TestRunnerParallelism(t *testing.T) {
	r, err := NewRunner("master")
	if err != nil {
		t.Fatal(err)
	}

	cfg := `
{
  "cluster_checks": {
    "check1": {
      "cmd": ["sleep", "1"],
      "timeout": "5s"
    },
    "check2": {
      "cmd": ["sleep", "1"],
      "timeout": "5s"
    },
    "check3": {
      "cmd": ["sleep", "1"],
      "timeout": "5s"
    },
    "check4": {
      "cmd": ["sleep", "1"],
      "timeout": "5s"
    },
    "check5": {
      "cmd": ["sleep", "1"],
      "timeout": "5s"
    }
  },
  "node_checks": {
    "checks": {
      "check1": {
        "cmd": ["sleep", "1"],
        "timeout": "5s"
      },
      "check2": {
        "cmd": ["sleep", "1"],
        "timeout": "5s"
      },
      "check3": {
        "cmd": ["sleep", "1"],
        "timeout": "5s"
      },
      "check4": {
        "cmd": ["sleep", "1"],
        "timeout": "5s"
      },
      "check5": {
        "cmd": ["sleep", "1"],
        "timeout": "5s"
      }
    },
    "prestart": ["check1", "check2", "check3", "check4", "check5"],
    "poststart": ["check1", "check2", "check3", "check4", "check5"]
  }
}`
	err = r.Load(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}

	// Each check should take 1 second, so we expect that parallalel check runs also take 1 second, with a 100 millisecond
	// buffer.
	maxDuration := (1 * time.Second) + (100 * time.Millisecond)

	err = runWithTimeout(maxDuration, func() {
		r.Cluster(context.TODO(), false)
	})
	if err != nil {
		t.Fatal(errors.Wrap(err, "cluster check parallelism test failed"))
	}

	err = runWithTimeout(maxDuration, func() {
		r.PreStart(context.TODO(), false)
	})
	if err != nil {
		t.Fatal(errors.Wrap(err, "node-prestart check parallelism test failed"))
	}

	err = runWithTimeout(maxDuration, func() {
		r.PostStart(context.TODO(), false)
	})
	if err != nil {
		t.Fatal(errors.Wrap(err, "node-poststart check parallelism test failed"))
	}
}
