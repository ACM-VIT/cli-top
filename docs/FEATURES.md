# CLI-TOP Features

CLI-TOP is a command-line interface for VIT's VTOP portal. Here are all the available commands and their usage:

## Global Flags
- `-d, --debug`: Print Debug Messages
- `-u, --update`: Check for Updates
- `-v, --version`: Print Version Number

## Commands

### Academic Information

#### Profile
```bash
cli-top profile
```
Shows your VTOP Student Profile information.

#### Marks
```bash
cli-top marks [-s SEMESTER]
```
Shows marks details for a particular semester.

#### Grades
```bash
cli-top grades [-s SEMESTER]
```
Shows grade details for a particular semester.

#### CGPA
```bash
cli-top cgpa
```
Shows your CGPA details.

### Course Management

#### Course Page
```bash
cli-top course-page [-s SEMESTER] [-c COURSE] [-f FACULTY] [-i FUZZY_INDEX]
```
Download course materials for a selected semester, course, and faculty.

#### Syllabus
```bash
cli-top syllabus [-c COURSE]
```
Download syllabus for a selected course.

#### Attendance
```bash
cli-top attendance [-s SEMESTER]
```
Shows attendance details for a particular semester.

#### Timetable
```bash
cli-top timetable [-s SEMESTER]
```
Shows time table for a particular semester.

#### Holiday
```bash
cli-top holiday [-s SEMESTER] [-g CLASS_GROUP]
```
Shows upcoming class-impacting holidays for a particular semester.

#### Today
```bash
cli-top today [-s SEMESTER] [-g CLASS_GROUP]
```
Shows today's effective schedule and whether your current attendance gives you room to skip each class.

#### Tomorrow
```bash
cli-top tomorrow [-s SEMESTER] [-g CLASS_GROUP]
```
Shows tomorrow's effective schedule and whether your current attendance gives you room to skip each class.

#### Day After
```bash
cli-top dayafter [-s SEMESTER] [-g CLASS_GROUP]
```
Shows the day after tomorrow's effective schedule and whether your current attendance gives you room to skip each class.

#### Calendar
```bash
cli-top calendar [-s SEMESTER] [-g CLASS_GROUP]
```
Shows calendar with class schedule.

### Examination

#### Exam Schedule
```bash
cli-top exams [-s SEMESTER]
```
Shows exam schedule for a particular semester.

### Campus Services

#### Facility
```bash
cli-top facility
```
View or register for physical facilities.

#### Hostel
```bash
cli-top hostel
```
Shows your hostel details.

#### Library Dues
```bash
cli-top library-dues
```
Shows your library dues.

#### Receipts
```bash
cli-top receipts
```
Shows your receipt details.

#### Nightslip
```bash
cli-top nightslip
```
Shows your nightslip request status first. If no request is pending, the command asks whether you want to apply and then walks you through the required details interactively before submission.

#### Leave
```bash
cli-top leave
```
Shows your leave request status first. If no request is pending, the command asks whether you want to apply and then walks you through the leave type, place, reason, dates, and times interactively before submission.

For deterministic automation, submit non-interactively with explicit flags:
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
The command validates the leave type locally against known VTOP codes, normalizes dates and times, checks the date/time range, rejects malformed place/reason text before opening the leave workflow, and verifies the leave type against VTOP's form before posting.

### Communication

#### Class Messages
```bash
cli-top class-messages
```
Shows class messages.

### Account Management

#### Logout
```bash
cli-top logout
```
Logs out from VTOP.

## Usage Examples

1. Check your current semester's attendance:
```bash
cli-top attendance
```

2. Download course materials for a specific semester:
```bash
cli-top course-page -s 1
```

3. View exam schedule:
```bash
cli-top exams
```

4. Check your grades for a specific semester:
```bash
cli-top grades -s 2
```

## Notes
- Most semester-specific commands will prompt for semester selection if the `-s` flag is not provided
- The course page download supports fuzzy search for easier course selection
- All commands require you to be logged in first
- Use `cli-top [command] --help` for more information about a specific command 
