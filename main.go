package main

import (
	"fmt"
	"os"
	"time"
)

const usage = `bamboo - BambooHR time tracking CLI

Usage:
  bamboo in          Clock in  (alias: clock-in)
  bamboo out         Clock out (alias: clock-out)
  bamboo st          Today's timesheet entries (alias: status)

Configuration:
  Set these environment variables (or use a .env file):
    BAMBOO_API_KEY      Your BambooHR API key
    BAMBOO_COMPANY      Your company subdomain (from https://XXX.bamboohr.com)
    BAMBOO_EMPLOYEE_ID  Your numeric employee ID`

func run(args []string) int {
	if len(args) < 2 {
		fmt.Println(usage)
		return 1
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %s\n", err)
		fmt.Fprintf(os.Stderr, "Copy .env.example to .env and fill in your values\n")
		return 1
	}

	client := NewClient(cfg)

	switch args[1] {
	case "in", "clock-in":
		if err := client.ClockIn(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		fmt.Println("Clocked in.")

	case "out", "clock-out":
		if err := client.ClockOut(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		fmt.Println("Clocked out.")

	case "st", "status":
		emp, err := client.GetEmployee()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}

		fmt.Printf("%s\n", emp.DisplayName)
		fmt.Printf("%s — %s", emp.JobTitle, emp.Department)
		if emp.Location != "" {
			fmt.Printf(" (%s)", emp.Location)
		}
		fmt.Println()
		fmt.Println()

		entries, err := client.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}

		if len(entries) == 0 {
			fmt.Println("No entries today.")
			return 0
		}

		var totalWorked time.Duration
		var clockedInSince string

		for _, e := range entries {
			start := parseTime(e.Start)
			if e.End == "" {
				clockedInSince = formatTime(start)
				elapsed := time.Since(start).Truncate(time.Minute)
				totalWorked += elapsed
				fmt.Printf("  %s - now  (%s)\n", formatTime(start), formatDuration(elapsed))
			} else {
				end := parseTime(e.End)
				dur := end.Sub(start)
				totalWorked += dur
				fmt.Printf("  %s - %s  (%s)\n", formatTime(start), formatTime(end), formatDuration(dur))
			}
			if e.Note != "" {
				fmt.Printf("           %s\n", e.Note)
			}
		}

		fmt.Println()
		if clockedInSince != "" {
			fmt.Printf("Clocked in since %s\n", clockedInSince)
		} else {
			fmt.Println("Clocked out")
		}
		fmt.Printf("Total today: %s\n", formatDuration(totalWorked))

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[1])
		fmt.Println(usage)
		return 1
	}

	return 0
}

func parseTime(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"15:04",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func formatTime(t time.Time) string {
	return t.Local().Format("15:04")
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func main() {
	os.Exit(run(os.Args))
}
