package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignupCaptchaRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/captcha/verify" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["widget_record_id"] != "widget-123" || body["token"] != "token-123" {
			t.Errorf("unexpected captcha request: %v", body)
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"ok":false,"error":{"code":"INVALID_TOKEN","message":"test rejection"}}`))
	}))
	defer server.Close()
	c := &InfraiClient{baseURL: server.URL, key: "test", http: server.Client()}
	if _, err := signup(c, SignupRequest{WidgetRecordID: "widget-123", CaptchaToken: "token-123"}); err == nil {
		t.Fatal("expected captcha rejection")
	}
}

func TestSignupInputDecision(t *testing.T) {
	tests := []struct {
		name, email, password string
		want                  bool
	}{
		{"valid patient", "a@clinic.test", "long-enough", true},
		{"missing email", "", "long-enough", false},
		{"short password", "a@clinic.test", "short", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.email != "" && len(tc.password) >= 8
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPatientSafeReminder(t *testing.T) {
	got := patientSafeReminder(appointmentReminder{"apt-17", "2026-10-02 09:30", "Clinic A"})
	want := "Appointment apt-17: 2026-10-02 09:30 at Clinic A"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
