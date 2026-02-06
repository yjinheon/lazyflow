# Lazyflow

Lazyflow is a terminal user interface for Airflow 3

## Installation

### Prerequisites

- Go 1.25 or higher

### Build from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/yjinheon/lazyflow.git
   cd lazyflow
   ```

2. Build the binary:
   ```bash
   go build -o lazyflow cmd/lazyflow/main.go
   ```

3. Run the application:
   ```bash
   ./lazyflow
   ```

## Keybindings

### Global

| Key | Action |
| --- | --- |
| Ctrl+C | Quit application |
| F5 | Refresh data |
| Esc | Focus DAG list |
| Tab | Cycle focus (DAG List -> Main View -> Back) |
| ? | Show help |
| / | Search |

### Navigation (Tabs)

| Key | View |
| --- | --- |
| 1 | Runs |
| 2 | Tasks |
| 3 | Logs |
| 4 | Code |
| 5 | Config |
| 6 | Connections |
| 7 | Variables |
| 8 | Monitor |
| 9 | Lineage |

### DAG List

| Key | Action |
| --- | --- |
| a | Filter: Active only |
| A | Filter: All |
| f | Filter: Failed only |
| d | Focus: DAG List |
| i | Focus: DAG Info |

## Acknowledgements

This project is inspired by k9s, kdash, and lazygit.

- https://github.com/derailed/k9s
- https://github.com/kdash-rs/kdash
- https://github.com/jesseduffield/lazygit