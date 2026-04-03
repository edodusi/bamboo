package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	cfg := &Config{
		APIKey:     "test-key",
		Company:    "test-co",
		EmployeeID: "42",
	}
	client := NewClient(cfg)
	client.BaseURL = srv.URL
	return client, srv
}

func TestClockIn_Success(t *testing.T) {
	client, srv := testClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/time_tracking/employees/42/clock_in") {
			t.Errorf("path = %s, want .../clock_in", r.URL.Path)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "test-key" || pass != "x" {
			t.Errorf("auth = (%q, %q, %v), want (test-key, x, true)", user, pass, ok)
		}

		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := client.ClockIn(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClockIn_Error(t *testing.T) {
	client, srv := testClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"not allowed"}`))
	})
	defer srv.Close()

	err := client.ClockIn()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestClockOut_Success(t *testing.T) {
	client, srv := testClient(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/time_tracking/employees/42/clock_out") {
			t.Errorf("path = %s, want .../clock_out", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if err := client.ClockOut(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetEmployee_Success(t *testing.T) {
	client, srv := testClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/employees/42") {
			t.Errorf("path = %s, want .../employees/42", r.URL.Path)
		}
		fields := r.URL.Query().Get("fields")
		if !strings.Contains(fields, "displayName") {
			t.Errorf("fields = %s, want displayName included", fields)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Employee{
			DisplayName: "Edoardo Dusi",
			JobTitle:    "DevRel Manager",
			Department:  "Engineering",
			Location:    "Remote",
		})
	})
	defer srv.Close()

	emp, err := client.GetEmployee()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emp.DisplayName != "Edoardo Dusi" {
		t.Errorf("DisplayName = %q, want %q", emp.DisplayName, "Edoardo Dusi")
	}
	if emp.JobTitle != "DevRel Manager" {
		t.Errorf("JobTitle = %q, want %q", emp.JobTitle, "DevRel Manager")
	}
}

func TestGetEmployee_Error(t *testing.T) {
	client, srv := testClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	})
	defer srv.Close()

	_, err := client.GetEmployee()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestStatus_Success(t *testing.T) {
	entries := []TimesheetEntry{
		{ID: 1, EmployeeID: 42, Date: "2026-04-03", Start: "09:00", End: "12:30"},
		{ID: 2, EmployeeID: 42, Date: "2026-04-03", Start: "13:30", End: ""},
	}

	client, srv := testClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/time_tracking/timesheet_entries") {
			t.Errorf("path = %s, want .../timesheet_entries", r.URL.Path)
		}
		if r.URL.Query().Get("employeeIds") != "42" {
			t.Errorf("employeeIds = %s, want 42", r.URL.Query().Get("employeeIds"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	})
	defer srv.Close()

	result, err := client.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}
	if result[0].Start != "09:00" {
		t.Errorf("first entry start = %q, want %q", result[0].Start, "09:00")
	}
	if result[1].End != "" {
		t.Errorf("second entry end = %q, want empty (still clocked in)", result[1].End)
	}
}

func TestStatus_Empty(t *testing.T) {
	client, srv := testClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	})
	defer srv.Close()

	result, err := client.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("got %d entries, want 0", len(result))
	}
}

func TestParseTime_RFC3339(t *testing.T) {
	got := parseTime("2026-04-03T09:15:00+02:00")
	if got.Hour() != 9 || got.Minute() != 15 {
		t.Errorf("parseTime RFC3339 = %v, want 09:15", got)
	}
}

func TestParseTime_Short(t *testing.T) {
	got := parseTime("14:30")
	if got.Hour() != 14 || got.Minute() != 30 {
		t.Errorf("parseTime short = %v, want 14:30", got)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{3*time.Hour + 30*time.Minute, "3h30m"},
		{45 * time.Minute, "45m"},
		{0, "0m"},
		{8 * time.Hour, "8h00m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestRun_NoArgs(t *testing.T) {
	code := run([]string{"bamboo"})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	noEnvFiles()
	t.Setenv("BAMBOO_API_KEY", "k")
	t.Setenv("BAMBOO_COMPANY", "c")
	t.Setenv("BAMBOO_EMPLOYEE_ID", "1")

	code := run([]string{"bamboo", "nope"})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
