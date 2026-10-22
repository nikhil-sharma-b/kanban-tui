package cli

const helpText = `kanban - keyboard-first kanban board

Usage:
  kanban                      open the TUI
  kanban <group> <command>    run one command and exit

Projects:
  kanban project list
  kanban project show [PROJECT]
  kanban project create NAME
  kanban project rename PROJECT NAME
  kanban project delete PROJECT --yes
  kanban project use PROJECT            set the default project

Tasks:
  kanban task list [-p PROJECT] [--column COLUMN] [--search TEXT] [--archived]
  kanban task show TASK
  kanban task add TITLE [-d DESCRIPTION] [-p PROJECT] [--column COLUMN]
  kanban task edit TASK [--title TITLE] [-d DESCRIPTION]
  kanban task move TASK [--column COLUMN] [--position N]
  kanban task archive TASK
  kanban task restore TASK
  kanban task delete TASK

Columns:
  kanban column list [-p PROJECT]
  kanban column add NAME [-p PROJECT]
  kanban column rename COLUMN NAME [-p PROJECT]
  kanban column move COLUMN --position N [-p PROJECT]
  kanban column delete COLUMN [-p PROJECT]

Daily board (columns: waiting, active, next):
  kanban daily show
  kanban daily add TITLE [-d DESCRIPTION] [--column COLUMN]
  kanban daily promote TASK --to PROJECT [--column COLUMN]

References:
  PROJECT  project name (case-insensitive) or ID; -p defaults to the active project
  TASK     task ID or any unique prefix of it, e.g. 3f2a1b9c
  COLUMN   column name or title, e.g. backlog, "in progress", in-progress
  N        1-based position; tasks default to the end of the column

Every command accepts --json and prints the affected task, project or board.
Errors go to stderr; exit code 1 for failures, 2 for bad usage.
`
