package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestConcurrentRequests fires many simultaneous requests through a real
// HTTP server and checks that every response matches its own request. Run
// with -race to also detect unsynchronised access to shared state.
func TestConcurrentRequests(t *testing.T) {
	srv := httptest.NewServer(newTestServer(t, nil))
	defer srv.Close()

	const workers = 200
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := range workers {
		go func() {
			defer wg.Done()

			// Every tenth request fails, so error paths run concurrently too.
			failing := i%10 == 0
			reqID := fmt.Sprintf("req-%d", i)
			body := fmt.Sprintf(`{"operation":"multiply","operands":[%d,3]}`, i)
			if failing {
				body = `{"operation":"divide","operands":[1,0]}`
			}

			req, err := http.NewRequest(http.MethodPost, srv.URL+calculationsPath, strings.NewReader(body))
			if err != nil {
				t.Error(err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(requestIDHeader, reqID)

			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			defer resp.Body.Close()

			if got := resp.Header.Get(requestIDHeader); got != reqID {
				t.Errorf("request %d: X-Request-ID = %q, want %q", i, got, reqID)
			}

			if failing {
				var p problem
				if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
					t.Errorf("request %d: decoding problem: %v", i, err)
					return
				}
				if resp.StatusCode != http.StatusUnprocessableEntity || p.Code != codeDivisionByZero {
					t.Errorf("request %d: got %d %s, want 422 %s", i, resp.StatusCode, p.Code, codeDivisionByZero)
				}
				return
			}

			var c calculationResponse
			if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
				t.Errorf("request %d: decoding result: %v", i, err)
				return
			}
			if want := float64(i * 3); resp.StatusCode != http.StatusOK || c.Result != want {
				t.Errorf("request %d: got %d result %v, want 200 result %v", i, resp.StatusCode, c.Result, want)
			}
		}()
	}

	wg.Wait()
}
