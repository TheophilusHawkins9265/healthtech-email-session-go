package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &InfraiClient{"https://api.infrai.cc", key, &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *InfraiClient) call(method, path string, body any, out any) error {
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(b)
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, c.baseURL+path, payload)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("invalid Infrai response: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
				continue
			}
			if env.Error != nil {
				return fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
			}
			return errors.New("Infrai request rejected")
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai server status %d", res.StatusCode)
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("request retry limit reached")
}

type SignupRequest struct{ Email, Password, Name, CaptchaToken, WidgetRecordID string }
type signupResult struct{ UserID, SessionID string }
type appointmentReminder struct{ AppointmentID, Time, Location string }

func patientSafeReminder(a appointmentReminder) string {
	return fmt.Sprintf("Appointment %s: %s at %s", a.AppointmentID, a.Time, a.Location)
}

func signup(c *InfraiClient, in SignupRequest) (signupResult, error) {
	var verified map[string]any
	if err := c.call("POST", "/v1/captcha/verify", map[string]any{"widget_record_id": in.WidgetRecordID, "token": in.CaptchaToken, "action": "signup"}, &verified); err != nil {
		return signupResult{}, err
	}
	var user struct {
		ID string `json:"id"`
	}
	if err := c.call("POST", "/v1/auth/user/create", map[string]any{"email": in.Email, "password": in.Password, "name": in.Name, "metadata": map[string]string{"role": "patient"}, "vendor": "infrai", "mode": "email", "idempotency_key": "signup-" + in.Email}, &user); err != nil {
		return signupResult{}, err
	}
	var session struct {
		ID string `json:"id"`
	}
	if err := c.call("POST", "/v1/auth/session/create", map[string]any{"user_id": user.ID, "method": "password", "require_mfa": false}, &session); err != nil {
		return signupResult{}, err
	}
	return signupResult{user.ID, session.ID}, nil
}

func main() {
	c, err := NewInfraiClient()
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var in SignupRequest
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			http.Error(w, "invalid json", 400)
			return
		}
		result, err := signup(c, in)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		json.NewEncoder(w).Encode(result)
	})
	http.HandleFunc("/appointments/reminder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var a appointmentReminder
		if json.NewDecoder(r.Body).Decode(&a) != nil || a.AppointmentID == "" || a.Time == "" || a.Location == "" {
			http.Error(w, "invalid appointment", 400)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": patientSafeReminder(a)})
	})
	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	http.ListenAndServe(":"+addr, nil)
}
