package backend

import "testing"

func TestNormalizeOneBrowserSystemProxy(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "Clash address", raw: "127.0.0.1:7890", want: "http://127.0.0.1:7890"},
		{name: "HTTP mapping", raw: "http=127.0.0.1:7890;https=127.0.0.1:7890", want: "http://127.0.0.1:7890"},
		{name: "HTTPS first", raw: "https=127.0.0.1:7891;http=127.0.0.1:7890", want: "http://127.0.0.1:7891"},
		{name: "SOCKS mapping", raw: "socks=127.0.0.1:7891", want: "socks5://127.0.0.1:7891"},
		{name: "SOCKS URL", raw: "socks5://127.0.0.1:7891", want: "socks5://127.0.0.1:7891"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeOneBrowserSystemProxy(tt.raw)
			if err != nil {
				t.Fatalf("normalize proxy: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeOneBrowserSystemProxyRejectsInvalid(t *testing.T) {
	for _, raw := range []string{"", "http=", "not-a-proxy"} {
		if _, err := normalizeOneBrowserSystemProxy(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}
