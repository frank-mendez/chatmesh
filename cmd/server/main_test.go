package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInstanceHandler(t *testing.T) {
	tests := []struct {
		serverID string
		wantBody string
	}{
		{"server-1", `{"server_id":"server-1"}`},
		{"server-2", `{"server_id":"server-2"}`},
	}
	for _, tt := range tests {
		t.Run(tt.serverID, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/instance", nil)
			rr := httptest.NewRecorder()
			instanceHandler(tt.serverID)(rr, req)
			if got := rr.Body.String(); got != tt.wantBody {
				t.Errorf("body got %q, want %q", got, tt.wantBody)
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type got %q, want application/json", ct)
			}
		})
	}
}
