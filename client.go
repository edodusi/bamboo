package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	Config  *Config
	BaseURL string
	HTTP    *http.Client
}

type Employee struct {
	DisplayName string `json:"displayName"`
	JobTitle    string `json:"jobTitle"`
	Department  string `json:"department"`
	Location    string `json:"location"`
}

type DirectoryEmployee struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	JobTitle    string `json:"jobTitle"`
	Supervisor  string `json:"supervisor"`
}

type TimeOffRequest struct {
	ID         string            `json:"id"`
	EmployeeID string            `json:"employeeId"`
	Start      string            `json:"start"`
	End        string            `json:"end"`
	Status     TimeOffStatus     `json:"status"`
	Type       TimeOffType       `json:"type"`
	Amount     TimeOffAmount     `json:"amount"`
	Dates      map[string]string `json:"dates"`
}

type TimeOffStatus struct {
	Status string `json:"status"`
}

type TimeOffType struct {
	Name string `json:"name"`
}

type TimeOffAmount struct {
	Unit   string `json:"unit"`
	Amount string `json:"amount"`
}

type TimesheetEntry struct {
	ID         int    `json:"id"`
	EmployeeID int    `json:"employeeId"`
	Date       string `json:"date"`
	Start      string `json:"start"`
	End        string `json:"end"`
	Note       string `json:"note"`
	ProjectID  int    `json:"projectId"`
	TaskID     int    `json:"taskId"`
}

func NewClient(cfg *Config) *Client {
	return &Client{
		Config:  cfg,
		BaseURL: fmt.Sprintf("https://%s.bamboohr.com/api/v1", cfg.Company),
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) doRequest(method, path string, payload ...any) ([]byte, int, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if len(payload) > 0 && payload[0] != nil {
		b, err := json.Marshal(payload[0])
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.SetBasicAuth(c.Config.APIKey, "x")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	return body, resp.StatusCode, nil
}

func (c *Client) GetEmployee() (*Employee, error) {
	path := fmt.Sprintf("/employees/%s?fields=displayName,jobTitle,department,location", c.Config.EmployeeID)
	body, status, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, apiError("get employee", status, body)
	}

	var emp Employee
	if err := json.Unmarshal(body, &emp); err != nil {
		return nil, fmt.Errorf("parsing employee: %w", err)
	}
	return &emp, nil
}

func (c *Client) ClockIn(at *time.Time) error {
	path := fmt.Sprintf("/time_tracking/employees/%s/clock_in", c.Config.EmployeeID)

	var payload any
	if at != nil {
		payload = map[string]string{
			"date":     at.Format("2006-01-02"),
			"start":    at.Format("15:04"),
			"timezone": ianaTimezone(),
		}
	}

	body, status, err := c.doRequest("POST", path, payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return apiError("clock in", status, body)
	}
	return nil
}

func (c *Client) ClockOut(at *time.Time) error {
	path := fmt.Sprintf("/time_tracking/employees/%s/clock_out", c.Config.EmployeeID)

	var payload any
	if at != nil {
		payload = map[string]string{
			"date":     at.Format("2006-01-02"),
			"end":      at.Format("15:04"),
			"timezone": ianaTimezone(),
		}
	}

	body, status, err := c.doRequest("POST", path, payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return apiError("clock out", status, body)
	}
	return nil
}

// apiError extracts a human-friendly message from BambooHR error responses.
// Falls back to the raw body if parsing fails.
func apiError(action string, statusCode int, raw []byte) error {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err == nil && parsed.Error.Message != "" {
		return fmt.Errorf("%s failed: %s", action, parsed.Error.Message)
	}
	return fmt.Errorf("%s failed (HTTP %d): %s", action, statusCode, string(raw))
}

func (c *Client) StatusRange(start, end string) ([]TimesheetEntry, error) {
	path := fmt.Sprintf("/time_tracking/timesheet_entries?employeeIds=%s&start=%s&end=%s",
		c.Config.EmployeeID, start, end)

	body, status, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, apiError("status check", status, body)
	}

	var entries []TimesheetEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return entries, nil
}

// ianaTimezone returns the IANA timezone name (e.g. "Europe/Rome").
// Falls back to UTC offset if the name can't be determined.
func ianaTimezone() string {
	zone := time.Now().Location().String()
	if zone != "Local" {
		return zone
	}
	// On macOS/Linux, /etc/localtime is a symlink to the zoneinfo file
	if target, err := os.Readlink("/etc/localtime"); err == nil {
		// e.g. /var/db/timezone/zoneinfo/Europe/Rome -> Europe/Rome
		if idx := strings.Index(target, "zoneinfo/"); idx != -1 {
			return target[idx+len("zoneinfo/"):]
		}
	}
	// Check TZ env var
	if tz := os.Getenv("TZ"); tz != "" {
		return tz
	}
	// Last resort: UTC offset
	_, offset := time.Now().Zone()
	return time.FixedZone("", offset).String()
}

// DirectReports returns employees whose `supervisor` field matches supervisorName.
// The directory stores supervisor as "FirstName LastName", matching Employee.DisplayName.
func (c *Client) DirectReports(supervisorName string) ([]DirectoryEmployee, error) {
	body, status, err := c.doRequest("GET", "/employees/directory")
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, apiError("directory", status, body)
	}

	var resp struct {
		Employees []DirectoryEmployee `json:"employees"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing directory: %w", err)
	}

	var reports []DirectoryEmployee
	for _, e := range resp.Employees {
		if e.Supervisor == supervisorName {
			reports = append(reports, e)
		}
	}
	return reports, nil
}

// TimeOffRequestsForEmployees returns approved time-off requests overlapping [start, end] for the given employees.
// Note: the API filters by request-level start/end overlap, so requests spanning the window are included.
func (c *Client) TimeOffRequestsForEmployees(employeeIDs []string, start, end string) ([]TimeOffRequest, error) {
	path := fmt.Sprintf("/time_off/requests/?start=%s&end=%s&status=approved", start, end)

	body, status, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, apiError("time off requests", status, body)
	}

	var all []TimeOffRequest
	if err := json.Unmarshal(body, &all); err != nil {
		return nil, fmt.Errorf("parsing time off: %w", err)
	}

	wanted := make(map[string]bool, len(employeeIDs))
	for _, id := range employeeIDs {
		wanted[id] = true
	}
	var filtered []TimeOffRequest
	for _, r := range all {
		if wanted[r.EmployeeID] {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// StatusRangeForEmployees queries timesheet entries for multiple employees in a single call.
func (c *Client) StatusRangeForEmployees(employeeIDs []string, start, end string) ([]TimesheetEntry, error) {
	ids := strings.Join(employeeIDs, ",")
	path := fmt.Sprintf("/time_tracking/timesheet_entries?employeeIds=%s&start=%s&end=%s", ids, start, end)

	body, status, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, apiError("team status", status, body)
	}

	var entries []TimesheetEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return entries, nil
}

func (c *Client) Status() ([]TimesheetEntry, error) {
	today := time.Now().Format("2006-01-02")
	path := fmt.Sprintf("/time_tracking/timesheet_entries?employeeIds=%s&start=%s&end=%s",
		c.Config.EmployeeID, today, today)

	body, status, err := c.doRequest("GET", path)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, apiError("status check", status, body)
	}

	var entries []TimesheetEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return entries, nil
}
