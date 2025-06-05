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

#### Nightslip Status
```bash
cli-top nightslip
```
Shows your nightslip request status.

#### Leave Status
```bash
cli-top leavestatus
```
Shows your leave request status.

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