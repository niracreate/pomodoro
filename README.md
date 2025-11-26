# Pomodoro CLI Timer

A simple, interactive, keyboard-driven Pomodoro timer for the terminal.

<p align="center">
  <img src="assets/Initial%20Screen.png" alt="Setup Screen" width="40%">
  <!--&nbsp; &nbsp;-->
  <img src="assets/timer.png" alt="Timer Running" width="45%">
</p>

## Features

- Clean, colorful terminal interface with ASCII timer display
- Interactive setup menu and quick-start via command-line arguments
- Pause, resume, skip, and adjust time on the fly
- Desktop notifications when sessions end (Windows, macOS, Linux)
- Sound alerts for session changes
- Fully customizable work/break durations and number of sessions

## Installation

### Option 1: Go Install

If you have Go installed, you can install it directly:

```bash
go install github.com/niracreate/pomo@latest
```

### Option 2: Build from Source

Clone the repo and build the binary.

```bash
git clone https://github.com/niracreate/pomo.git
cd pomo
go build -o pomo main.go
```

## Usage

### 1. Interactive Mode

Simply run the tool without arguments to enter the setup screen:

```bash
pomo
```

### 2. Quick Start (CLI Arguments)

Skip the setup and start the timer immediately.

**Syntax:** `pomo [work] [break] [sessions]`

```bash
# Start 25m work (defaults: 5m break, 4 sessions)
pomo 25m

# Start 50m work, 10m break
pomo 50m 10m

# Start 45m work, 15m break, 6 sessions
pomo 45m 15m 6
```

## Controls

### Setup Screen

| Key                 | Action        |
| :------------------ | :------------ |
| `TAB`/`Mouse wheel` | Switch inputs |
| `ENTER`             | Start Timer   |
| `q`                 | Quit          |

### Timer Screen

| Key       | Action                   |
| :-------- | :----------------------- |
| `SPACE`   | Pause / Resume           |
| `s`       | **Skip** current session |
| `↑` / `↓` | +/- 1 minute             |
| `q`       | Quit                     |

### Built With

- **[Go](https://go.dev/)**
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** — TUI Framework
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** — Styling
- **[Beeep](https://github.com/gen2brain/beeep)** — Notifications
