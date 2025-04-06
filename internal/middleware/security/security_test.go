package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicSecurity(t *testing.T) {
	handler := BasicSecurity(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, response.Code)
	}
}
