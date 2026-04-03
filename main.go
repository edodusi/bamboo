package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const usage = `bamboo - BambooHR time tracking CLI

Usage:
  bamboo in [TIME]   Clock in now or at TIME (alias: clock-in)
  bamboo out [TIME]  Clock out now or at TIME (alias: clock-out)
  bamboo st          Today's timesheet entries (alias: status)
  bamboo w           This week's summary (alias: week)
  bamboo lw          Last week's summary (alias: last-week)
  bamboo m           This month's summary (alias: month)
  bamboo lm          Last month's summary (alias: last-month)

  TIME formats: 9am, 9:00am, 9 am, 9:00 am, 9:00, 17:30

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
		at, err := parseTimeArg(args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		if err := client.ClockIn(at); err != nil {
			if strings.Contains(err.Error(), "CLOCKED_IN") {
				since := clockedInSince(client)
				if since != "" {
					fmt.Printf("Already clocked in since %s.\n", since)
				} else {
					fmt.Println("Already clocked in.")
				}
				return 1
			}
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		if at != nil {
			fmt.Printf("Clocked in at %s.\n", at.Format("15:04"))
		} else {
			fmt.Println("Clocked in.")
		}

	case "out", "clock-out":
		at, err := parseTimeArg(args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		if err := client.ClockOut(at); err != nil {
			if strings.Contains(err.Error(), "CLOCKED_OUT") {
				fmt.Println("Already clocked out.")
				return 1
			}
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		if at != nil {
			fmt.Printf("Clocked out at %s.\n", at.Format("15:04"))
		} else {
			fmt.Println("Clocked out.")
		}

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

	case "week", "w":
		start, end := weekRange(time.Now(), 0)
		return showRange(client, "This Week", start, end)

	case "last-week", "lw":
		start, end := weekRange(time.Now(), -1)
		return showRange(client, "Last Week", start, end)

	case "month", "m":
		start, end := monthRange(time.Now(), 0)
		return showRange(client, time.Now().Format("January 2006"), start, end)

	case "last-month", "lm":
		prev := time.Now().AddDate(0, -1, 0)
		start, end := monthRange(prev, 0)
		return showRange(client, prev.Format("January 2006"), start, end)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[1])
		fmt.Println(usage)
		return 1
	}

	return 0
}

func weekRange(now time.Time, offset int) (string, string) {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday-1)+(offset*7))
	friday := monday.AddDate(0, 0, 4)
	return monday.Format("2006-01-02"), friday.Format("2006-01-02")
}

func monthRange(ref time.Time, _ int) (string, string) {
	start := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, ref.Location())
	end := start.AddDate(0, 1, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func showRange(client *Client, label, start, end string) int {
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
	fmt.Printf("\n%s (%s → %s)\n\n", label, start, end)

	entries, err := client.StatusRange(start, end)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return 1
	}

	if len(entries) == 0 {
		fmt.Println("No entries.")
		return 0
	}

	// Group by date
	days := make(map[string][]TimesheetEntry)
	var dayOrder []string
	for _, e := range entries {
		date := e.Date
		if date == "" {
			// Extract date from Start if Date is empty
			t := parseTime(e.Start)
			if !t.IsZero() {
				date = t.Local().Format("2006-01-02")
			}
		}
		if _, ok := days[date]; !ok {
			dayOrder = append(dayOrder, date)
		}
		days[date] = append(days[date], e)
	}

	var grandTotal time.Duration

	for _, date := range dayOrder {
		dayEntries := days[date]
		var dayTotal time.Duration

		t, _ := time.Parse("2006-01-02", date)
		dayLabel := t.Format("Mon Jan 2")

		for _, e := range dayEntries {
			start := parseTime(e.Start)
			if e.End == "" {
				elapsed := time.Since(start).Truncate(time.Minute)
				dayTotal += elapsed
			} else {
				end := parseTime(e.End)
				dayTotal += end.Sub(start)
			}
		}

		grandTotal += dayTotal
		fmt.Printf("  %-12s  %s\n", dayLabel, formatDuration(dayTotal))
	}

	fmt.Printf("\n  %-12s  %s\n", "Total", formatDuration(grandTotal))
	return 0
}

func clockedInSince(client *Client) string {
	entries, err := client.Status()
	if err != nil {
		return ""
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].End == "" {
			return formatTime(parseTime(entries[i].Start))
		}
	}
	return ""
}

// parseTimeArg parses optional time arguments like ["9am"], ["9:00", "am"], ["17:30"].
// Returns nil if no args provided.
func parseTimeArg(args []string) (*time.Time, error) {
	if len(args) == 0 {
		return nil, nil
	}

	raw := strings.ToLower(strings.Join(args, ""))
	raw = strings.ReplaceAll(raw, " ", "")

	layouts := []string{
		"3:04pm",  // 9:00am
		"3:04 pm", // shouldn't hit after space removal, but safe
		"3pm",     // 9am
		"15:04",   // 17:30
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			now := time.Now()
			result := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			return &result, nil
		}
	}

	return nil, fmt.Errorf("could not parse time %q (try: 9am, 9:00am, 9:30 am, 17:30)", strings.Join(args, " "))
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
