package main

import (
	"fmt"
	"os"
)

const usage = `bamboo - BambooHR time tracking CLI

Usage:
  bamboo in   Clock in
  bamboo out  Clock out
  bamboo st   Show today's timesheet entries

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
	case "in":
		if err := client.ClockIn(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		fmt.Println("Clocked in.")

	case "out":
		if err := client.ClockOut(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		fmt.Println("Clocked out.")

	case "st", "status":
		entries, err := client.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return 1
		}
		if len(entries) == 0 {
			fmt.Println("No entries today.")
			return 0
		}
		for _, e := range entries {
			end := e.End
			if end == "" {
				end = "..."
			}
			note := ""
			if e.Note != "" {
				note = fmt.Sprintf("  (%s)", e.Note)
			}
			fmt.Printf("%s  %s - %s%s\n", e.Date, e.Start, end, note)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[1])
		fmt.Println(usage)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run(os.Args))
}
