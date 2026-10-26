package proxy

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestSpeedProbeUsesOneRequestAndFallsBackFromHEAD(t *testing.T) {
	for _, headAllowed := range []bool{true, false} {
		t.Run(map[bool]string{true: "HEAD", false: "GET fallback"}[headAllowed], func(t *testing.T) {
			var heads, gets atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					heads.Add(1)
					if !headAllowed {
						w.WriteHeader(405)
						return
					}
				} else {
					gets.Add(1)
				}
				w.WriteHeader(204)
			}))
			defer server.Close()
			_, status, err := doSpeedTestRequest(server.Client(), server.URL)
			if err != nil || status != 204 {
				t.Fatalf("status=%d err=%v", status, err)
			}
			wantGets := int32(0)
			if !headAllowed {
				wantGets = 1
			}
			if heads.Load() != 1 || gets.Load() != wantGets {
				t.Fatalf("HEAD=%d GET=%d", heads.Load(), gets.Load())
			}
		})
	}
}
