package proxy

import "testing"

func TestNormalizeIPHealthPayloadFromNestedLocation(t *testing.T) {
	payload := map[string]interface{}{
		"ip": "203.0.113.7",
		"location": map[string]interface{}{
			"country":      "Germany",
			"country_code": "DE",
			"state":        "Berlin",
			"city":         "Berlin",
		},
	}
	if err := normalizeIPHealthPayload(payload, "ipapi.is"); err != nil {
		t.Fatalf("normalize payload: %v", err)
	}
	if got := mapString(payload, "country"); got != "Germany" {
		t.Fatalf("country = %q", got)
	}
	if got := mapString(payload, "region"); got != "Berlin" {
		t.Fatalf("region = %q", got)
	}
	if got := mapString(payload, "countryCode"); got != "DE" {
		t.Fatalf("countryCode = %q", got)
	}
}

func TestNormalizeIPHealthPayloadRejectsHTMLLikeEmptyResponse(t *testing.T) {
	payload := map[string]interface{}{"message": "<!DOCTYPE html>"}
	if err := normalizeIPHealthPayload(payload, "ipwho.is"); err == nil {
		t.Fatal("expected missing IP response to fail")
	}
}

func TestNormalizeIPInfoCountryAsCode(t *testing.T) {
	payload := map[string]interface{}{"ip": "203.0.113.8", "country": "DE", "region": "Berlin", "city": "Berlin"}
	if err := normalizeIPHealthPayload(payload, "ipinfo.io"); err != nil {
		t.Fatalf("normalize payload: %v", err)
	}
	if got := mapString(payload, "countryCode"); got != "DE" {
		t.Fatalf("countryCode = %q", got)
	}
}
