# cli-top command reference

Use `cli-top --help` to list public commands and `cli-top COMMAND --help` to inspect a command.

## Global flags

| Flag | Effect |
| --- | --- |
| `-d`, `--debug` | Enable debug output. |
| `-h`, `--help` | Show help. |
| `-u`, `--update` | Check for updates when used as `cli-top --update`. |
| `-v`, `--version` | Print the installed version when used as `cli-top --version`. |

## Account

### Login

```bash
cli-top login [--username USERNAME] [--password PASSWORD]
```

Stores the username and encrypted password in cli-top's local configuration. If either value is omitted, cli-top prompts for it. This command does not open a VTOP session; the next portal-backed command authenticates with the stored credentials.

### Logout

```bash
cli-top logout
```

Clears stored VTOP credentials and session data.

## Academic records

| Command | Flags | Purpose |
| --- | --- | --- |
| `cli-top profile` | — | Show the student profile. |
| `cli-top marks` | `-s`, `--semester INDEX` | Show marks for a semester. |
| `cli-top grades` | `-s`, `--semester INDEX` | Show grades for a semester. |
| `cli-top cgpa` | — | Show registered and earned credits, CGPA, and grade counts. |
| `cli-top attendance` | `-s`, `--semester INDEX` | Show attendance and skip/attend guidance. |
| `cli-top exams` | `-s`, `--semester INDEX` | Show the upcoming exam schedule. |

Without `--semester`, `attendance` and `exams` search recent semesters for available data. Commands with an interactive semester picker prompt when no selection is supplied.

## Schedule and planning

| Command | Flags | Purpose |
| --- | --- | --- |
| `cli-top timetable` | `-s`, `--semester INDEX` | Show a semester timetable. |
| `cli-top holiday` | `-s`, `--semester INDEX`; `-g`, `--class-group INDEX` | Show upcoming class-impacting holidays. |
| `cli-top today` | `-s`, `--semester INDEX`; `-g`, `--class-group INDEX` | Show today's effective schedule and attendance-aware skip guidance. |
| `cli-top tomorrow` | `-s`, `--semester INDEX`; `-g`, `--class-group INDEX` | Show tomorrow's effective schedule and attendance-aware skip guidance. |
| `cli-top dayafter` | `-s`, `--semester INDEX`; `-g`, `--class-group INDEX` | Show the same planner for the day after tomorrow. Alias: `day-after`. |
| `cli-top calendar` | `-s`, `--semester INDEX`; `-g`, `--class-group INDEX` | Show the academic calendar with class scheduling. |

## Courses and downloads

| Command | Flags | Purpose |
| --- | --- | --- |
| `cli-top course-allocation` | `--category QUERY_OR_INDEX`; `-c`, `--course QUERY_OR_INDEX` | Browse course-allocation details. |
| `cli-top course-page` | `-s`, `--semester INDEX`; `-c`, `--course INDEX`; `-f`, `--faculty QUERY_OR_INDEX`; `--materials LIST` | Download materials from the current consolidated course page. |
| `cli-top course-page-archive` | `-s`, `--semester INDEX`; `-c`, `--course INDEX`; `-f`, `--faculty QUERY_OR_INDEX`; `--materials LIST` | Download materials through the older course-page workflow. |
| `cli-top syllabus` | `-c`, `--course QUERY_OR_INDEX` | Download a course syllabus. |
| `cli-top da` | `-c`, `--course QUERY_OR_INDEX`; `-a`, `--assignment QUERY_OR_INDEX` | Review digital-assignment deadlines and submission status, and download available question papers. |

For `--materials`, use comma-separated indices and ranges such as `1,2-4`, or `0` for all materials. Omit selection flags to use the interactive pickers.

## Campus services

| Command | Flags | Purpose |
| --- | --- | --- |
| `cli-top events` | — | View upcoming club events and interactively register for an open event. |
| `cli-top facility` | `-f`, `--facility QUERY_OR_INDEX`; `--confirm` | View or register for a physical facility. |
| `cli-top hostel` | — | Show hostel details. |
| `cli-top library-dues` | — | Show library dues. |
| `cli-top receipts` | — | Show receipts and payment history. |
| `cli-top nightslip` | — | Show nightslip status and, when none is pending, optionally apply interactively. |
| `cli-top leave` | See below. | Show leave status and, when none is pending, optionally apply interactively. |

Submit a leave request non-interactively with all required application fields:

```bash
cli-top leave --apply \
  --leave-code HT1 \
  --visiting-place "Chennai" \
  --reason "Family visit" \
  --from-date 2026-04-23 \
  --from-time 20:30 \
  --to-date 2026-04-24 \
  --to-time 06:30
```

The leave application flags are `--apply`, `--leave-code`, `--visiting-place`, `--reason`, `--from-date`, `--from-time`, `--to-date`, and `--to-time`. Application fields are rejected unless `--apply` is present.

## Communication

```bash
cli-top msg
cli-top class-messages
```

Both forms show class messages. `msg` is the canonical command and `class-messages` is its alias.

## Shell completion

```bash
cli-top completion SHELL
```

Supported shells are `bash`, `fish`, `powershell`, and `zsh`.

All portal-backed feature commands require stored credentials. See the project [README](../README.md) for installation and first-use examples.
