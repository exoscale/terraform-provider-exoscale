package kms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
)

// testZone is the only zone known to the mock API.
const testZone = "ch-gva-2"

// mockSuccess is the response of the KMS operations that only report success.
const mockSuccess = `{"status":"success"}`

// mockProblem is an error response of the KMS API: every client error is a 400 with a problem type.
func mockProblem(problemType, title string) string {
	return fmt.Sprintf(`{"type":%q,"title":%q,"status":400,"detail":"no details."}`, problemType, title)
}

type mockResponse struct {
	status int
	body   string
}

// mockAPI is a mock of the Exoscale API. It serves the responses expected for each route
// ("METHOD /path") in order, records the calls and their JSON bodies, and fails the test on
// an unexpected call. The zone lookups made to switch the client zone are answered but not recorded.
type mockAPI struct {
	t      *testing.T
	server *httptest.Server

	mu        sync.Mutex
	responses map[string][]mockResponse
	calls     []string
	bodies    map[string][]map[string]any
}

func newMockAPI(t *testing.T) *mockAPI {
	t.Helper()

	m := &mockAPI{
		t:         t,
		responses: map[string][]mockResponse{},
		bodies:    map[string][]map[string]any{},
	}
	m.server = httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(m.server.Close)

	return m
}

// expect queues a response for a route.
func (m *mockAPI) expect(route string, status int, body string) *mockAPI {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responses[route] = append(m.responses[route], mockResponse{status: status, body: body})
	return m
}

// client returns an API client using the mock.
func (m *mockAPI) client() *exoscale.Client {
	m.t.Helper()

	client, err := exoscale.NewClient(
		credentials.NewStaticCredentials("EXOtest", "secret"),
		exoscale.ClientOptWithEndpoint(exoscale.Endpoint(m.server.URL)),
	)
	if err != nil {
		m.t.Fatal(err)
	}

	return client
}

// assertCalls checks the routes called, in order.
func (m *mockAPI) assertCalls(want ...string) {
	m.t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	if fmt.Sprint(m.calls) != fmt.Sprint(want) {
		m.t.Errorf("got calls %v, want %v", m.calls, want)
	}
}

// requestBody returns the JSON body of the n-th call to a route.
func (m *mockAPI) requestBody(route string, n int) map[string]any {
	m.t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.bodies[route]) <= n {
		m.t.Fatalf("no body for call #%d to %s", n, route)
	}
	return m.bodies[route][n]
}

func (m *mockAPI) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet && r.URL.Path == "/zone" {
		_, _ = fmt.Fprintf(w, `{"zones":[{"name":%q,"api-endpoint":%q}]}`, testZone, m.server.URL)
		return
	}

	route := r.Method + " " + r.URL.Path

	m.mu.Lock()
	m.calls = append(m.calls, route)
	if data, _ := io.ReadAll(r.Body); len(data) > 0 {
		var body map[string]any
		if err := json.Unmarshal(data, &body); err == nil {
			m.bodies[route] = append(m.bodies[route], body)
		}
	}
	queue := m.responses[route]
	if len(queue) == 0 {
		m.mu.Unlock()
		m.t.Errorf("unexpected call to %s", route)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := queue[0]
	m.responses[route] = queue[1:]
	m.mu.Unlock()

	w.WriteHeader(resp.status)
	_, _ = w.Write([]byte(resp.body))
}
