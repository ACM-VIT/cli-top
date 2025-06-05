![ACM Header](https://user-images.githubusercontent.com/14032427/92643737-e6252e00-f2ff-11ea-8a51-1f1b69caba9f.png)

<h1 align="center">cli-top</h1>

<p align="center">
  <strong>A Command Line Interface (CLI) tool for seamless interaction with the student portal, VTOP.</strong>
</p>

<p align="center">
  <a href="https://acmvit.in/" target="_blank">
    <img alt="Made by ACM" src="https://img.shields.io/badge/MADE%20BY-ACM%20VIT-blue?style=for-the-badge"/>
  </a>
  <!-- Uncomment the below line to add the license badge. Make sure the right license badge is reflected. -->
  <!-- <img alt="license" src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" /> -->
</p>

---

## Overview

**cli-top** is an easy-to-use tool for VIT students that helps them quickly access important information from the VTOP student portal. Whether it's checking grades, viewing the timetable, or handling assignments, cli-top makes it simple to get what you need.



## Features

- **Login**: Secure login to the VTOP portal
- **Mark View**: Check your marks for various courses
- **Digital Assignment**: Manage your digital assignment submissions
- **Course Page**: Access course materials and updates
- **Academic Calendar**: Keep track of important academic dates
- **Exam Schedule**: View upcoming exam schedules
- **Attendance Calculator**: Calculate your attendance percentage
- **Time Table**: Easily view your class schedule
- **Class Messages**: Stay updated with class announcements
- **Leave Status**: Check the status of your leave applications
- **Nightslip Status**: Monitor your hostel nightslip requests
- **Library Dues**: Stay on top of library dues
- **Receipts**: Access fee receipts and payment history
- **Grade View**: Review your grades and academic performance
- **Student Profile**: View personal details
- **Hostel Info**: Check hostel details 
- **CGPA View**: Track your cumulative GPA over semesters
- **Syllabus**: Easily download syllabus files
- **Facility**: View hostel facilities
- **Logout**: Securely logout from the CLI



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

After installation, you can access various features of **cli-top** by running specific commands:

- **Login to VTOP:**

```bash
./cli-top login 
```

- **View Marks:**

```bash
./cli-top marks
```
- **Calculate Attendance:**
```bash
./cli-top attendance
```
- For a full list of commands and features of cli-top, you can run:

```bash
./cli-top help
```


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
- [Harshit Vootukuri](https://github.com/hvoot36)
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


