package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

func TestOneBrowserStartupCleanupOnlyClosesInternalTabs(t *testing.T) {
	for _, sole := range []bool{false, true} {
		t.Run(strconv.FormatBool(sole), func(t *testing.T) {
			targets := []map[string]string{{"id": "ntp", "type": "page", "url": "chrome://newtab/"}, {"id": "apps", "type": "page", "url": "chrome://apps/"}, {"id": "worker", "type": "service_worker", "url": "chrome://newtab/"}}
			if !sole {
				targets = append(targets, map[string]string{"id": "personal", "type": "page", "url": "https://example.com/newtab/"})
			}
			var mu sync.Mutex
			methods := []string{}
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/json/list":
					_ = json.NewEncoder(w).Encode(targets)
				case "/json/version":
					_ = json.NewEncoder(w).Encode(map[string]string{"webSocketDebuggerUrl": "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"})
				case "/ws":
					conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
					if err != nil {
						return
					}
					defer conn.Close()
					var request struct {
						ID     int            `json:"id"`
						Method string         `json:"method"`
						Params map[string]any `json:"params"`
					}
					if err := conn.ReadJSON(&request); err != nil {
						return
					}
					mu.Lock()
					if request.Method == "Target.closeTarget" {
						methods = append(methods, "close:"+request.Params["targetId"].(string))
					} else {
						methods = append(methods, request.Method)
					}
					mu.Unlock()
					_ = conn.WriteJSON(map[string]any{"id": request.ID, "result": map[string]any{"success": true, "targetId": "replacement"}})
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			port, _ := strconv.Atoi(strings.TrimPrefix(server.URL, "http://127.0.0.1:"))
			if err := cleanOneBrowserStartupTabs(port); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"close:ntp", "close:apps"}
			if sole {
				want = append([]string{"Target.createTarget"}, want...)
			}
			if strings.Join(methods, ",") != strings.Join(want, ",") {
				t.Fatalf("cleanup sequence=%v, want %v", methods, want)
			}
		})
	}
	for _, url := range []string{"https://example.com/chrome://newtab/", "chrome-extension://id/newtab.html", "about:blank", "https://chromewebstore.google.com/"} {
		if oneBrowserUnwantedStartupTab(url) {
			t.Fatal("personal tab classified as internal", url)
		}
	}
}

func TestOneBrowserSavedSessionDoesNotRequestExtraWindow(t *testing.T) {
	args := oneBrowserRestoreWindowArgs([]string{"--user-data-dir=private", "--new-window", "--restore-last-session"})
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--new-window") || !strings.Contains(joined, "--restore-last-session") || !strings.Contains(joined, "--user-data-dir=private") {
		t.Fatal(args)
	}
	launch, deferred, _ := buildBrowserLaunchTargets([]string{"file:///workspace.html", "https://www.google.com/"}, nil, true, false, false)
	if len(launch) != 2 || len(deferred) != 0 {
		t.Fatal("fresh profile must open requested pages directly")
	}
	launch, deferred, _ = buildBrowserLaunchTargets(nil, nil, true, true, false)
	if len(launch) != 0 || len(deferred) != 0 {
		t.Fatal("saved session must not append startup tabs")
	}
}
