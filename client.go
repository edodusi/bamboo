package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		return nil, fmt.Errorf("get employee failed (HTTP %d): %s", status, string(body))
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
			"timezone": at.Location().String(),
		}
	}

	body, status, err := c.doRequest("POST", path, payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("clock in failed (HTTP %d): %s", status, string(body))
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
			"timezone": at.Location().String(),
		}
	}

	body, status, err := c.doRequest("POST", path, payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("clock out failed (HTTP %d): %s", status, string(body))
	}
	return nil
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
		return nil, fmt.Errorf("status check failed (HTTP %d): %s", status, string(body))
	}

	var entries []TimesheetEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return entries, nil
}
