![ACM Header](https://user-images.githubusercontent.com/14032427/92643737-e6252e00-f2ff-11ea-8a51-1f1b69caba9f.png)

<h1 align="center">cli-top</h1>

<p align="center">
  <strong>A Command Line Interface (CLI) tool for seamless interaction with the student portal, VTOP.</strong>
</p>

<p align="center">
  <a href="https://acmvit.in/" target="_blank">
    <img alt="Made by ACM" src="https://img.shields.io/badge/MADE%20BY-ACM%20VIT-blue?style=for-the-badge"/>
  </a>
  <a href="LICENSE">
    <img alt="License: GPL v3" src="https://img.shields.io/badge/License-GPLv3-blue.svg?style=for-the-badge"/>
  </a>
</p>

---

## Overview

**cli-top** gives VIT students quick terminal access to VTOP records, schedules, course resources, and campus services.

## Features

- **Academic records:** attendance, marks, grades, CGPA, exam schedules, and profile details
- **Planning:** timetable, academic calendar, holidays, and attendance-aware plans for today, tomorrow, and the day after
- **Course tools:** course allocation, current and archived course materials, syllabi, and digital-assignment status and question papers
- **Campus services:** events, facilities, hostel details, leave, nightslips, library dues, and receipts
- **Communication:** class messages and announcements
- **Local account management:** encrypted credential storage and logout

See the [command reference](docs/FEATURES.md) for every command, alias, and flag.

## Tech Stack

- **GoLang** : Core programming language
- **Cobra** : Go library for creating the terminal CLI

## Installation

To install **cli-top**, you can download the binary directly from [cli-top.acmvit.in](https://cli-top.acmvit.in/).

1. **Download the Binary:**

   Visit [cli-top.acmvit.in](https://cli-top.acmvit.in/) to download and configure the appropriate binary for your operating system.

2. **Run the Binary:**

After downloading, navigate to the folder where the binary is saved and run it from your terminal:

```bash
./cli-top
```

### Build from Source

To compile **cli-top** yourself:

```bash
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o cli-top .
```

Using `-trimpath` removes local file paths from the executable and, together with `-ldflags "-s -w"`, helps reduce binary size.

## Usage

Store your VTOP credentials locally. Missing values are prompted for interactively:

```bash
cli-top login [--username USERNAME] [--password PASSWORD]
```

`login` encrypts and stores the password; the first portal-backed command performs the VTOP authentication.

Run a feature command:

```bash
cli-top marks
cli-top attendance --semester 1
cli-top exams --semester 1
cli-top today
cli-top msg # alias: class-messages
```

List all public commands or inspect one command's flags:

```bash
cli-top --help
cli-top attendance --help
```

## Testing

```bash
go test ./...
go test -race ./...
```

Cross-package workflow and executable tests live in [`tests/`](tests/). Package-local `_test.go` files are reserved for white-box unit tests that exercise unexported implementation details.

## Project Management

- Use **Git** for version control and code management
- Take up issues and request assignment before starting work
- Push to the "dev" branch for testing and compatibility checks
- Main and dev branch pushes require approval from designated maintainers

## Authors

- [Saharsh Bhansali](https://github.com/saharshbhansali)
- [Manav Muthanna](https://github.com/ManavMuthanna)
- [Sarthak Gupta](https://github.com/gptsarthak)

## Maintainers

- [Garv Jain](https://github.com/notcoolgarv)
- [Tanmay Paturu](https://github.com/Tintedfireglass)
- [Shambhavi Paygude](https://github.com/shambhavipaygude)
- [Harshit Vootukuri](https://github.com/btcry)
- [Adheesh Garg](https://github.com/qwerty-dvorak)
- [Ishaan S](https://github.com/theg1239)

## Contributors

- [Prateek Srivastava](https://github.com/prateek-srivastava001)
- [Kaustav Patro](https://github.com/icky-kp)
- [Amritsai](https://github.com/gekyxme)
- [Pritam Satpathy](https://github.com/ps2181)
- [Kaustubh Kanodia](https://github.com/Quasar-025)
- [Rohit Sakamuri](https://github.com/rohitphaniramsakamuri)
- [Mahendra Choudhary](https://github.com/mahendra785)
- [Drashti Shukla](https://github.com/drashtishukla)
- [Shreyas Mishra](https://github.com/ShreyasM09)
- [Aditya Naik](https://github.com/Zxcivic)
- [Yash Sinha](https://github.com/yashsinha1224)
- [Aditya Singh](https://github.com/adii2ma)
- [Vansh Bhatiya](https://github.com/bhatiyavansh)
- [Yashika Panda](https://github.com/yashikaa2005)
- [Shruthilaya K](https://github.com/shruthilayak11)
- [Srijan Srivastava](https://github.com/Srijan1202)

## License

Copyright (C) 2023-2026 ACM-VIT and cli-top contributors.

This project is licensed under the GNU General Public License version 3 only
(`GPL-3.0-only`). See [LICENSE](LICENSE) for the full license text.
