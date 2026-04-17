package main

import (
	"fmt"
	"os"
	"sort"
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
  bamboo team [P]    Direct reports summary for period P (alias: t)
                     P: w (default), lw, m, lm

  TIME formats: 9am, 9:00am, 9 am, 9:00 am, 9:00, 14, 17:30

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

	case "team", "t":
		period := "w"
		if len(args) >= 3 {
			period = args[2]
		}
		label, start, end, ok := periodRange(period)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown period: %s (use w, lw, m, lm)\n", period)
			return 1
		}
		return showTeamRange(client, label, start, end)

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

	timeOff, err := client.TimeOffRequestsForEmployees([]string{client.Config.EmployeeID}, start, end)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch time off: %s\n", err)
	}
	pto := make(map[string]string)
	for _, r := range timeOff {
		for date, amt := range r.Dates {
			unit := "d"
			if r.Amount.Unit == "hours" {
				unit = "h"
			}
			pto[date] = fmt.Sprintf("%s (%s%s)", r.Type.Name, amt, unit)
		}
	}

	if len(entries) == 0 && len(pto) == 0 {
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
	for date := range pto {
		if _, ok := days[date]; !ok {
			dayOrder = append(dayOrder, date)
		}
	}
	sort.Strings(dayOrder)

	var grandTotal time.Duration
	ptoDays := 0

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
		line := fmt.Sprintf("  %-12s  %s", dayLabel, formatDuration(dayTotal))
		if ptoNote, ok := pto[date]; ok {
			line += "  [" + ptoNote + "]"
			ptoDays++
		}
		fmt.Println(line)
	}

	summary := fmt.Sprintf("\n  %-12s  %s", "Total", formatDuration(grandTotal))
	if ptoDays > 0 {
		summary += fmt.Sprintf("  (+%d PTO)", ptoDays)
	}
	fmt.Println(summary)
	return 0
}

func periodRange(period string) (label, start, end string, ok bool) {
	switch period {
	case "w", "week":
		s, e := weekRange(time.Now(), 0)
		return "This Week", s, e, true
	case "lw", "last-week":
		s, e := weekRange(time.Now(), -1)
		return "Last Week", s, e, true
	case "m", "month":
		s, e := monthRange(time.Now(), 0)
		return time.Now().Format("January 2006"), s, e, true
	case "lm", "last-month":
		prev := time.Now().AddDate(0, -1, 0)
		s, e := monthRange(prev, 0)
		return prev.Format("January 2006"), s, e, true
	}
	return "", "", "", false
}

func showTeamRange(client *Client, label, start, end string) int {
	me, err := client.GetEmployee()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return 1
	}

	reports, err := client.DirectReports(me.DisplayName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return 1
	}
	if len(reports) == 0 {
		fmt.Printf("No direct reports found for %s.\n", me.DisplayName)
		return 0
	}

	ids := make([]string, 0, len(reports))
	for _, r := range reports {
		ids = append(ids, r.ID)
	}

	entries, err := client.StatusRangeForEmployees(ids, start, end)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return 1
	}

	timeOff, err := client.TimeOffRequestsForEmployees(ids, start, end)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch time off: %s\n", err)
	}

	byEmp := make(map[int][]TimesheetEntry)
	for _, e := range entries {
		byEmp[e.EmployeeID] = append(byEmp[e.EmployeeID], e)
	}

	// ptoByEmp[empID][date] = "Type (amount unit)"
	ptoByEmp := make(map[string]map[string]string)
	for _, r := range timeOff {
		if _, ok := ptoByEmp[r.EmployeeID]; !ok {
			ptoByEmp[r.EmployeeID] = make(map[string]string)
		}
		for date, amt := range r.Dates {
			unit := "d"
			if r.Amount.Unit == "hours" {
				unit = "h"
			}
			ptoByEmp[r.EmployeeID][date] = fmt.Sprintf("%s (%s%s)", r.Type.Name, amt, unit)
		}
	}

	fmt.Printf("Team — %s (%s → %s)\n", label, start, end)

	for _, r := range reports {
		fmt.Println()
		fmt.Printf("%s — %s\n", r.DisplayName, r.JobTitle)

		empID := 0
		fmt.Sscanf(r.ID, "%d", &empID)
		empEntries := byEmp[empID]
		pto := ptoByEmp[r.ID]

		days := make(map[string][]TimesheetEntry)
		var dayOrder []string
		for _, e := range empEntries {
			date := e.Date
			if date == "" {
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
		// Include PTO-only days (no clocked entries) in the output
		for date := range pto {
			if _, ok := days[date]; !ok {
				dayOrder = append(dayOrder, date)
			}
		}
		sort.Strings(dayOrder)

		if len(dayOrder) == 0 {
			fmt.Println("  No entries.")
			continue
		}

		var total time.Duration
		ptoDays := 0
		for _, date := range dayOrder {
			var dayTotal time.Duration
			for _, e := range days[date] {
				s := parseTime(e.Start)
				if e.End == "" {
					dayTotal += time.Since(s).Truncate(time.Minute)
				} else {
					dayTotal += parseTime(e.End).Sub(s)
				}
			}
			total += dayTotal
			t, _ := time.Parse("2006-01-02", date)
			label := t.Format("Mon Jan 2")
			line := fmt.Sprintf("  %-12s  %s", label, formatDuration(dayTotal))
			if ptoNote, ok := pto[date]; ok {
				line += "  [" + ptoNote + "]"
				ptoDays++
			}
			fmt.Println(line)
		}
		summary := fmt.Sprintf("  %-12s  %s", "Total", formatDuration(total))
		if ptoDays > 0 {
			summary += fmt.Sprintf("  (+%d PTO)", ptoDays)
		}
		fmt.Println(summary)
	}
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
		"15",      // 14, 9 (bare hour, 24h)
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			now := time.Now()
			result := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			return &result, nil
		}
	}

	return nil, fmt.Errorf("could not parse time %q (try: 9am, 9:00am, 9:30 am, 14, 17:30)", strings.Join(args, " "))
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
