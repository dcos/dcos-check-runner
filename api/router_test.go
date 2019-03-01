package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/dcos/dcos-check-runner/runner"
	"github.com/pkg/errors"
)

func TestBaseURI(t *testing.T) {
	// Resources are hosted at the base URI.
	baseURIs := []string{"", "/", "/foo", "/foo/", "/foo/bar", "/_", "/@"}
	resources := []string{"/node/", "/cluster/"}
	for _, baseURI := range baseURIs {
		s, err := newTestServer("master", baseURI)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		for _, resource := range resources {
			// Check that the resource is at the base URI.
			if sc := getResponse(t, "GET", s.URL+baseURI+resource, nil, nil).StatusCode; sc != http.StatusOK {
				t.Fatalf("Expected status %d, got %d", http.StatusOK, sc)
			}
			// Check that the resource is not at a different base URI.
			if sc := getResponse(t, "GET", s.URL+"/bad_base_uri"+resource, nil, nil).StatusCode; sc != http.StatusNotFound {
				t.Fatalf("Expected status %d, got %d", http.StatusNotFound, sc)
			}
		}
	}
}

func TestAPI(t *testing.T) {
	// Create a master and agent server and test listing and running checks on each of them, decode responses, and
	// compare with expected decoded responses.
	master, err := newTestServer("master", "")
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	agent, err := newTestServer("agent", "")
	if err != nil {
		t.Fatal(err)
	}
	defer agent.Close()

	// Checks can be listed in groups by type, either node or cluster.
	t.Run("get node checks (master)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "GET", master.URL+"/node/", nil, nil), http.StatusOK, map[string]interface{}{
			"node-check": map[string]interface{}{
				"description": "Node check",
				"cmd":         interfaceSlice([]string{"echo", "node-check"}),
				"timeout":     "1s",
			},
			"node-check-master": map[string]interface{}{
				"description": "Node check master",
				"cmd":         interfaceSlice([]string{"echo", "node-check-master"}),
				"timeout":     "1s",
			},
		})
	})
	t.Run("get node checks (agent)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "GET", agent.URL+"/node/", nil, nil), http.StatusOK, map[string]interface{}{
			"node-check": map[string]interface{}{
				"description": "Node check",
				"cmd":         interfaceSlice([]string{"echo", "node-check"}),
				"timeout":     "1s",
			},
			"node-check-agent": map[string]interface{}{
				"description": "Node check agent",
				"cmd":         interfaceSlice([]string{"echo", "node-check-agent"}),
				"timeout":     "1s",
			},
		})
	})
	t.Run("get cluster checks (master)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "GET", master.URL+"/cluster/", nil, nil), http.StatusOK, map[string]interface{}{
			"cluster-check-1": map[string]interface{}{
				"description": "Cluster check 1",
				"cmd":         interfaceSlice([]string{"echo", "cluster-check-1"}),
				"timeout":     "1s",
			},
			"cluster-check-2": map[string]interface{}{
				"description": "Cluster check 2",
				"cmd":         interfaceSlice([]string{"echo", "cluster-check-2"}),
				"timeout":     "1s",
			},
		})
	})
	t.Run("get cluster checks (agent)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "GET", agent.URL+"/cluster/", nil, nil), http.StatusOK, map[string]interface{}{
			"cluster-check-1": map[string]interface{}{
				"description": "Cluster check 1",
				"cmd":         interfaceSlice([]string{"echo", "cluster-check-1"}),
				"timeout":     "1s",
			},
			"cluster-check-2": map[string]interface{}{
				"description": "Cluster check 2",
				"cmd":         interfaceSlice([]string{"echo", "cluster-check-2"}),
				"timeout":     "1s",
			},
		})
	})

	// Checks can be filtered with query parameters.
	t.Run("get node checks filtered by query param (master)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "GET", master.URL+"/node/?check=node-check", nil, nil), http.StatusOK, map[string]interface{}{
			"node-check": map[string]interface{}{
				"description": "Node check",
				"cmd":         interfaceSlice([]string{"echo", "node-check"}),
				"timeout":     "1s",
			},
		})
		assertJSONResponse(t, getResponse(t, "GET", master.URL+"/node/?check=node-check&check=node-check-master", nil, nil), http.StatusOK, map[string]interface{}{
			"node-check": map[string]interface{}{
				"description": "Node check",
				"cmd":         interfaceSlice([]string{"echo", "node-check"}),
				"timeout":     "1s",
			},
			"node-check-master": map[string]interface{}{
				"description": "Node check master",
				"cmd":         interfaceSlice([]string{"echo", "node-check-master"}),
				"timeout":     "1s",
			},
		})
	})
	t.Run("get cluster checks filtered by query param (agent)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "GET", agent.URL+"/cluster/?check=cluster-check-1", nil, nil), http.StatusOK, map[string]interface{}{
			"cluster-check-1": map[string]interface{}{
				"description": "Cluster check 1",
				"cmd":         interfaceSlice([]string{"echo", "cluster-check-1"}),
				"timeout":     "1s",
			},
		})
		assertJSONResponse(t, getResponse(t, "GET", agent.URL+"/cluster/?check=cluster-check-1&check=cluster-check-2", nil, nil), http.StatusOK, map[string]interface{}{
			"cluster-check-1": map[string]interface{}{
				"description": "Cluster check 1",
				"cmd":         interfaceSlice([]string{"echo", "cluster-check-1"}),
				"timeout":     "1s",
			},
			"cluster-check-2": map[string]interface{}{
				"description": "Cluster check 2",
				"cmd":         interfaceSlice([]string{"echo", "cluster-check-2"}),
				"timeout":     "1s",
			},
		})
	})

	// Checks can be run by group.
	t.Run("run node checks (master)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/node/", nil, nil), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
				"node-check-master": map[string]interface{}{
					"status": float64(0),
					"output": "node-check-master\n",
				},
			},
		})
	})
	t.Run("run node checks (agent)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/node/", nil, nil), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
				"node-check-agent": map[string]interface{}{
					"status": float64(0),
					"output": "node-check-agent\n",
				},
			},
		})
	})
	t.Run("run cluster checks (master)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/cluster/", nil, nil), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
				"cluster-check-2": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-2\n",
				},
			},
		})
	})
	t.Run("run cluster checks (agent)", func(t *testing.T) {
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/cluster/", nil, nil), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
				"cluster-check-2": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-2\n",
				},
			},
		})
	})

	// Check runs can be filtered by form.
	t.Run("run node checks filtered by form (master)", func(t *testing.T) {
		body := url.Values{"check": []string{"node-check"}}.Encode()
		headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/node/", headers, strings.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
			},
		})

		body = url.Values{"check": []string{"node-check", "node-check-master"}}.Encode()
		headers = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/node/", headers, strings.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
				"node-check-master": map[string]interface{}{
					"status": float64(0),
					"output": "node-check-master\n",
				},
			},
		})
	})
	t.Run("run node checks filtered by form (agent)", func(t *testing.T) {
		body := url.Values{"check": []string{"node-check"}}.Encode()
		headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/node/", headers, strings.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
			},
		})

		body = url.Values{"check": []string{"node-check", "node-check-agent"}}.Encode()
		headers = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/node/", headers, strings.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
				"node-check-agent": map[string]interface{}{
					"status": float64(0),
					"output": "node-check-agent\n",
				},
			},
		})
	})
	t.Run("run cluster checks filtered by form (master)", func(t *testing.T) {
		body := url.Values{"check": []string{"cluster-check-1"}}.Encode()
		headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/cluster/", headers, strings.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
			},
		})

		body = url.Values{"check": []string{"cluster-check-1", "cluster-check-2"}}.Encode()
		headers = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/cluster/", headers, strings.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
				"cluster-check-2": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-2\n",
				},
			},
		})
	})
	t.Run("run cluster checks filtered by form (agent)", func(t *testing.T) {
		body := url.Values{"check": []string{"cluster-check-1"}}.Encode()
		headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/cluster/", headers, strings.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
			},
		})

		body = url.Values{"check": []string{"cluster-check-1", "cluster-check-2"}}.Encode()
		headers = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/cluster/", headers, strings.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
				"cluster-check-2": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-2\n",
				},
			},
		})
	})

	// Check runs can be filtered by JSON.
	t.Run("run node checks filtered by JSON (master)", func(t *testing.T) {
		body, err := json.Marshal(map[string]interface{}{"check": []string{"node-check"}})
		if err != nil {
			t.Fatal(err)
		}
		headers := map[string]string{"Content-Type": "application/json; charset=utf-8"}
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/node/", headers, bytes.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
			},
		})

		body, err = json.Marshal(map[string]interface{}{"check": []string{"node-check", "node-check-master"}})
		if err != nil {
			t.Fatal(err)
		}
		headers = map[string]string{"Content-Type": "application/json; charset=utf-8"}
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/node/", headers, bytes.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
				"node-check-master": map[string]interface{}{
					"status": float64(0),
					"output": "node-check-master\n",
				},
			},
		})
	})
	t.Run("run node checks filtered by JSON (agent)", func(t *testing.T) {
		body, err := json.Marshal(map[string]interface{}{"check": []string{"node-check"}})
		if err != nil {
			t.Fatal(err)
		}
		headers := map[string]string{"Content-Type": "application/json; charset=utf-8"}
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/node/", headers, bytes.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
			},
		})

		body, err = json.Marshal(map[string]interface{}{"check": []string{"node-check", "node-check-agent"}})
		if err != nil {
			t.Fatal(err)
		}
		headers = map[string]string{"Content-Type": "application/json; charset=utf-8"}
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/node/", headers, bytes.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"status": float64(0),
					"output": "node-check\n",
				},
				"node-check-agent": map[string]interface{}{
					"status": float64(0),
					"output": "node-check-agent\n",
				},
			},
		})
	})
	t.Run("run cluster checks filtered by JSON (master)", func(t *testing.T) {
		body, err := json.Marshal(map[string]interface{}{"check": []string{"cluster-check-1"}})
		if err != nil {
			t.Fatal(err)
		}
		headers := map[string]string{"Content-Type": "application/json; charset=utf-8"}
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/cluster/", headers, bytes.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
			},
		})

		body, err = json.Marshal(map[string]interface{}{"check": []string{"cluster-check-1", "cluster-check-2"}})
		if err != nil {
			t.Fatal(err)
		}
		headers = map[string]string{"Content-Type": "application/json; charset=utf-8"}
		assertJSONResponse(t, getResponse(t, "POST", master.URL+"/cluster/", headers, bytes.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
				"cluster-check-2": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-2\n",
				},
			},
		})
	})
	t.Run("run cluster checks filtered by JSON (agent)", func(t *testing.T) {
		body, err := json.Marshal(map[string]interface{}{"check": []string{"cluster-check-1"}})
		if err != nil {
			t.Fatal(err)
		}
		headers := map[string]string{"Content-Type": "application/json; charset=utf-8"}
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/cluster/", headers, bytes.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
			},
		})

		body, err = json.Marshal(map[string]interface{}{"check": []string{"cluster-check-1", "cluster-check-2"}})
		if err != nil {
			t.Fatal(err)
		}
		headers = map[string]string{"Content-Type": "application/json; charset=utf-8"}
		assertJSONResponse(t, getResponse(t, "POST", agent.URL+"/cluster/", headers, bytes.NewReader(body)), http.StatusOK, map[string]interface{}{
			"status": float64(0),
			"checks": map[string]interface{}{
				"cluster-check-1": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-1\n",
				},
				"cluster-check-2": map[string]interface{}{
					"status": float64(0),
					"output": "cluster-check-2\n",
				},
			},
		})
	})
}

func TestAPIErrors(t *testing.T) {
	s, err := newTestServer("master", "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	t.Run("nonexistent resources", func(t *testing.T) {
		for _, resource := range []string{"/foo/", "/@", "/cluster/nonexistent/", "/node/_"} {
			if sc := getResponse(t, "GET", s.URL+resource, nil, nil).StatusCode; sc != http.StatusNotFound {
				t.Fatalf("Expected status %d, got %d", http.StatusNotFound, sc)
			}
		}
	})

	t.Run("bad Content-Type", func(t *testing.T) {
		body, err := json.Marshal(map[string]interface{}{"check": []string{"node-check"}})
		if err != nil {
			t.Fatal(err)
		}
		// Unsupported content types.
		for _, ct := range []string{"text/plain; charset=utf-8", "text/html", "application/pdf", "application/javascript", "json"} {
			headers := map[string]string{"Content-Type": ct}
			if sc := getResponse(t, "POST", s.URL+"/node/", headers, bytes.NewReader(body)).StatusCode; sc != http.StatusUnsupportedMediaType {
				fmt.Println(ct)
				t.Fatalf("Expected status %d, got %d", http.StatusUnsupportedMediaType, sc)
			}
		}
		// Invalid content types.
		for _, ct := range []string{"@", "/", "\\", "\\foo", ";"} {
			headers := map[string]string{"Content-Type": ct}
			if sc := getResponse(t, "POST", s.URL+"/node/", headers, bytes.NewReader(body)).StatusCode; sc != http.StatusBadRequest {
				fmt.Println(ct)
				t.Fatalf("Expected status %d, got %d", http.StatusBadRequest, sc)
			}
		}
	})

	t.Run("nonexistant check", func(t *testing.T) {
		if sc := getResponse(t, "GET", s.URL+"/node?check=foo", nil, nil).StatusCode; sc != http.StatusNotFound {
			t.Fatalf("Expected status %d, got %d", http.StatusNotFound, sc)
		}
	})

	t.Run("existing and nonexistant check", func(t *testing.T) {
		if sc := getResponse(t, "GET", s.URL+"/node?check=node-check-master&check=foo", nil, nil).StatusCode; sc != http.StatusNotFound {
			t.Fatalf("Expected status %d, got %d", http.StatusNotFound, sc)
		}
	})
}

// interfaceSlice returns a []interface{} initialized from strings.
func interfaceSlice(strings []string) []interface{} {
	interfaces := make([]interface{}, len(strings))
	for i, s := range strings {
		interfaces[i] = s
	}
	return interfaces
}

// newTestServer returns a *http.Server initialized with test check config.
func newTestServer(role string, baseURI string) (*httptest.Server, error) {
	r, err := runner.NewRunner(role)
	if err != nil {
		return nil, err
	}

	cfgJSON, err := json.Marshal(map[string]map[string]interface{}{
		"cluster_checks": map[string]interface{}{
			"cluster-check-1": map[string]interface{}{
				"description": "Cluster check 1",
				"cmd":         []string{"echo", "cluster-check-1"},
				"timeout":     "1s",
			},
			"cluster-check-2": map[string]interface{}{
				"description": "Cluster check 2",
				"cmd":         []string{"echo", "cluster-check-2"},
				"timeout":     "1s",
			},
		},
		"node_checks": map[string]interface{}{
			"checks": map[string]interface{}{
				"node-check": map[string]interface{}{
					"description": "Node check",
					"cmd":         []string{"echo", "node-check"},
					"timeout":     "1s",
				},
				"node-check-master": map[string]interface{}{
					"description": "Node check master",
					"cmd":         []string{"echo", "node-check-master"},
					"timeout":     "1s",
					"roles":       []string{"master"},
				},
				"node-check-agent": map[string]interface{}{
					"description": "Node check agent",
					"cmd":         []string{"echo", "node-check-agent"},
					"timeout":     "1s",
					"roles":       []string{"agent"},
				},
			},
			// The API doesn't provide prestart checks, so we don't need to define any.
			"prestart":  []string{},
			"poststart": []string{"node-check", "node-check-master", "node-check-agent"},
		},
	})
	if err != nil {
		return nil, err
	}

	if err := r.Load(strings.NewReader(string(cfgJSON[:]))); err != nil {
		return nil, err
	}

	return httptest.NewServer(NewRouter(r, baseURI)), nil
}

// getResponse executes a request and returns the response, failing the test if there is an error.
func getResponse(t *testing.T, method, url string, headers map[string]string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// assertJSONResponse fails the test if r is not a JSON response with the expected status and body.
func assertJSONResponse(t *testing.T, r *http.Response, statusCode int, expectedObj map[string]interface{}) {
	// Check status code.
	if r.StatusCode != statusCode {
		t.Fatalf("expected status %d, got %d", statusCode, r.StatusCode)
	}

	// Check Content-Type header.
	expectedCT := "application/json; charset=utf-8"
	ct, ok := r.Header["Content-Type"]
	if !ok {
		t.Fatal("No Content-Type header found in response")
	}
	if len(ct) != 1 {
		t.Fatalf("expected 1 Content-Type header, got %d", len(ct))
	}
	if ct[0] != expectedCT {
		t.Fatalf("expected Content-Type \"%s\", got \"%s\"", expectedCT, ct[0])
	}

	// Check body.
	var actualObj map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&actualObj); err != nil {
		t.Fatal(errors.Wrap(err, "Unable to decode response body"))
	}
	if !reflect.DeepEqual(expectedObj, actualObj) {
		fmt.Println("Expected:\n", expectedObj)
		fmt.Println("Actual:\n", actualObj)
		t.Fatal("Unexpected response")
	}
}
