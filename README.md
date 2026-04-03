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
bamboo in   # Clock in
bamboo out  # Clock out
bamboo st   # Show today's entries
```

### Shell aliases

Add to your `~/.zshrc`:

```bash
alias bi="bamboo in"
alias bo="bamboo out"
alias bs="bamboo st"
```

## License

MIT
