# bamboo

Minimal CLI for BambooHR time tracking. Clock in, clock out, check status — from your terminal.

## Install

```bash
go install github.com/edodusi/bamboo@latest
```

Or build from source:

```bash
git clone https://github.com/edodusi/bamboo.git
cd bamboo
go build -o bamboo .
```

## Setup

1. Generate an API key in BambooHR: Account → API Keys
2. Create a `.env` file (or export env vars):

```bash
cp .env.example .env
# Edit .env with your values
```

| Variable | Description |
|----------|-------------|
| `BAMBOO_API_KEY` | Your BambooHR API key |
| `BAMBOO_COMPANY` | Company subdomain (from `https://XXX.bamboohr.com`) |
| `BAMBOO_EMPLOYEE_ID` | Your numeric employee ID |

## Usage

```bash
bamboo in        # Clock in now
bamboo in 14     # Clock in at 14:00
bamboo in 9am    # Clock in at 9:00
bamboo out       # Clock out now
bamboo out 17:30 # Clock out at 17:30
bamboo st        # Show today's entries
bamboo w         # This week's summary
bamboo lw        # Last week's summary
bamboo m         # This month's summary
bamboo lm        # Last month's summary
bamboo team      # Direct reports — this week (alias: t)
bamboo team lw   # Direct reports — last week (also: m, lm)
```

The `team` command fetches direct reports from BambooHR at runtime (filtered by supervisor). No extra configuration required — your API key must have visibility into their timesheets.

Weekly and monthly summaries (`w`, `lw`, `m`, `lm`, `team`) annotate days with approved time-off pulled from BambooHR — Vacation, Sickness, Public Holiday, Work Travel, Nursing, etc. — so the totals are easy to reconcile against expected working days.

```
Mon Apr 6     0m       [Public Holiday (1d)]
Thu Apr 9     9h30m    [Work Travel (1d)]
Fri Apr 17    38m      [Nursing (4h)]
Total         35h35m   (+2 PTO)
```

Time formats: `9am`, `9:00am`, `9 am`, `9:00`, `14`, `17:30`

### Shell aliases

Add to your `~/.zshrc`:

```bash
alias bi="bamboo in"
alias bo="bamboo out"
alias bs="bamboo st"
alias bw="bamboo w"
```

## License

MIT
