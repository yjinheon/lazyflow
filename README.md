# Lazyflow

Lazyflow is a k9s-style terminal user interface for **Apache Airflow 3**, written in
Go using [`rivo/tview`](https://github.com/rivo/tview) + [`gdamore/tcell`](https://github.com/gdamore/tcell).
It talks to Airflow's REST API v2 and manages JWT auth internally.

## Features

- **Cluster overview** — a KPI bar at the top shows cluster-wide DAG counts
  (Active / Paused / Running / Success / Failed) over a configurable rollup window.
- **DAG list** with live filtering (active / all / failed) and search.
- **Drill-down navigation** — DAG → run → task → logs, each in its own tab.
- **Nine+ tabs**: Runs, Tasks, Logs, Code, Lineage, Monitor, Backfills,
  Connections, Variables, Config, plus a Help keymap page.
- **Gantt & lineage graph** toggles for the Tasks and Lineage tabs.
- **DAG actions** — trigger, pause/unpause, and backfill straight from the UI.
- **Backfill management** — pause, unpause, and cancel running backfills.
- **Cluster / pool panel** with a compact-vs-table view toggle.
- **Auto-refresh** with per-resource intervals; manual refresh on demand.

## Installation

### Prerequisites

- Go 1.25 or higher
- (Optional) [`just`](https://github.com/casey/just) as the task runner

### Build from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/yjinheon/lazyflow.git
   cd lazyflow
   ```

2. Build the binary:
   ```bash
   just build         # or: go build -o lazyflow ./cmd/lazyflow/
   ```

3. Run the application:
   ```bash
   just run           # or: ./lazyflow
   ```

Common `just` tasks: `build`, `run`, `dev` (build + run), `test`, `lint`, `tidy`, `clean`.

## Configuration

Config is loaded in this precedence order (later sources win):

1. `configs/default.yaml` (project-local), if present
2. `~/.config/lazyflow/config.yaml`
3. Environment overrides (always win):
   `AIRFLOW_BASE_URL`, `AIRFLOW_USERNAME`, `AIRFLOW_PASSWORD`,
   `AIRFLOW_TOKEN` (setting a token forces auth type to `token`)

Example `configs/default.yaml`:

```yaml
airflow:
  base_url: 'http://localhost:28080'
  timeout: '30s'
  auth:
    type: basic            # "basic" or "token"
    username: 'airflow'
    password: 'airflow'

ui:
  theme: dark
  refresh_intervals:
    dags: '5s'
    runs: '3s'
    tasks: '2s'
    logs: '1s'
    health: '10s'
  # Lookback window for the cluster KPI bar + per-DAG run counts.
  # Go duration (max unit 'h'); 7 days = 168h, 14 days = 336h.
  rollup_window: '168h'
```

A runtime debug log is written to `lazyflow.log` in the working directory
(recreated on each launch).

## Layout

```
┌ Header ─────────────────────────────────────────────┐
├ KPI Bar (cluster DAG counts) ───────────────────────┤
├ DAG List │ DAG Info │ Cluster / Pool Info ───────────┤
├ Tab Bar ────────────────────────────────────────────┤
├ Active Tab (Runs / Tasks / Logs / … / Help) ────────┤
├ Status Bar ─────────────────────────────────────────┘
```

## Keybindings

### Global

| Key | Action |
| --- | --- |
| Ctrl+C | Quit |
| F5 | Refresh |
| Esc | Close modal / execution view, or return focus to DAG list |
| Tab / Shift+Tab | Cycle panels: DAG list → info → run-filter → cluster → active tab |
| Left / Right | Previous / next tab |
| / | Search DAGs |
| ? | Show help keymap |

### Tabs

| Key | View |
| --- | --- |
| 1 | Runs |
| 2 | Tasks |
| 3 | Logs |
| 4 | Code |
| 5 | Lineage |
| 6 | Monitor |
| 7 | Backfills |
| 8 | Connections |
| 9 | Variables |
| 0 | Config |
| B | Backfills (alias) |
| g | Toggle Tasks gantt / Lineage graph |

### Navigation

| Key | Action |
| --- | --- |
| j / k | Move up / down |
| Enter | Select / drill down |

### DAG Actions

| Key | Action |
| --- | --- |
| t | Trigger selected DAG run |
| p | Pause / unpause selected DAG |
| b | Backfill selected DAG |

### Backfill Actions (Backfills tab)

| Key | Action |
| --- | --- |
| p / u | Pause / unpause selected backfill |
| c | Cancel selected backfill |

### DAG Filters

| Key | Action |
| --- | --- |
| a | Active DAGs only |
| A | All DAGs |
| f | Failed DAGs only |

### Focus

| Key | Action |
| --- | --- |
| d | Focus DAG list |
| i | Focus DAG info run-filter (↑↓ select, Enter filters Runs) |
| o | Focus cluster panel (press again to toggle pool compact/table) |

### Modal Actions

| Key | Action |
| --- | --- |
| Esc | Close without running |
| Enter | Submit when focused outside a JSON text area |
| Ctrl+J / Ctrl+M | Submit from anywhere in the form |

## Acknowledgements

This project is inspired by k9s, kdash, and lazygit.

- https://github.com/derailed/k9s
- https://github.com/kdash-rs/kdash
- https://github.com/jesseduffield/lazygit
