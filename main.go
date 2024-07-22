package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

// Status represents the status of a Git repository.
type Status struct {
	Commit    string
	Branch    string
	Upstream  string
	Ahead     int
	Behind    int
	Staged    int
	Conflict  int
	Modified  int
	Untracked int
	Stashed   int
}

// State represents the state of a Git repository during a specific operation.
type State struct {
	Step  int
	Total int
	State string
}

const (
	RebaseApply       string = "REBASE"
	RebaseMerge              = "REBASE-m"
	RebaseInteractive        = "REBASE-i"
	Am                       = "AM"
	AmRebase                 = "AM/REBASE"
	Merging                  = "MERGING"
	CherryPick               = "CHERRY-PICKING"
	Reverting                = "REVERTING"
	Bisecting                = "BISECTING"
)

// Symbols represents the symbols used to display the Git repository status.
type Symbols struct {
	Prefix    string
	Suffix    string
	Sep       string
	Local     string
	Ahead     string
	Behind    string
	Staged    string
	Conflict  string
	Modified  string
	Untracked string
	Stashed   string
	Clean     string
}

type Flags struct {
	Path    string
	Symbols Symbols
}

// main is the entry point of the program.
func main() {
	flags := Flags{Symbols: Symbols{}}
	flag.StringVar(&flags.Path, "path", "", "Path to the git repository. Leave empty for CWD.")
	flag.StringVar(&flags.Symbols.Prefix, "prefix", "[", "Prefix symbol")
	flag.StringVar(&flags.Symbols.Suffix, "suffix", "]", "Suffix symbol")
	flag.StringVar(&flags.Symbols.Sep, "sep", "|", "Separator symbol")
	flag.StringVar(&flags.Symbols.Local, "local", "L", "Local branch symbol")
	flag.StringVar(&flags.Symbols.Modified, "modified", "✚ ", "Modified symbol")
	flag.StringVar(&flags.Symbols.Staged, "staged", "● ", "Staged symbol")
	flag.StringVar(&flags.Symbols.Conflict, "conflict", "✖ ", "Conflict symbol")
	flag.StringVar(&flags.Symbols.Untracked, "untracked", "…", "Untracked symbol")
	flag.StringVar(&flags.Symbols.Stashed, "stashed", "⚑ ", "Stashed symbol")
	flag.StringVar(&flags.Symbols.Ahead, "ahead", "↑·", "Ahead symbol")
	flag.StringVar(&flags.Symbols.Behind, "behind", "↓·", "Behind symbol")
	flag.StringVar(&flags.Symbols.Clean, "clean", "✔", "Clean symbol")
	flag.Parse()

	state, err := gitState(flags.Path)
	if err != nil {
		log.Fatal(err)
	}

	if state == nil {
		// Nil state means not in a git repository
		return
	}

	output, err := gitStatus(flags.Path)
	if err != nil {
		log.Fatal(err)
	}

	status, err := parseStatus(output)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(buildOutput(*status, *state, flags.Symbols))
}

// gitState retrieves the current state of the Git repository.
func gitState(path string) (*State, error) {
	stdout, err := exec.Command(
		"git",
		"-C",
		path,
		"rev-parse",
		"--show-toplevel",
	).Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if e.ExitCode() == 128 {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("run cmd: %w", err)
	}

	if err := os.Chdir(strings.TrimSpace(string(stdout))); err != nil {
		return nil, fmt.Errorf("chdir: %w", err)
	}

	state := &State{State: ""}
	switch {
	case pathExists(".git/rebase-merge"):
		step, err := readInt(".git/rebase-merge/msgnum")
		if err != nil {
			return nil, fmt.Errorf("read rebase-merge/msgnum: %w", err)
		}
		state.Step = step

		total, err := readInt(".git/rebase-merge/end")
		if err != nil {
			return nil, fmt.Errorf("read rebase-merge/end: %w", err)
		}
		state.Total = total

		if pathExists(".git/rebase-merge/interactive") {
			state.State = RebaseInteractive
		} else {
			state.State = RebaseMerge
		}
	case pathExists(".git/rebase-apply"):
		step, err := readInt(".git/rebase-apply/next")
		if err != nil {
			return nil, fmt.Errorf("read rebase-apply/next: %w", err)
		}
		state.Step = step

		total, err := readInt(".git/rebase-apply/last")
		if err != nil {
			return nil, fmt.Errorf("read rebase-apply/last: %w", err)
		}
		state.Total = total

		switch {
		case pathExists(".git/rebase-apply/rebasing"):
			state.State = RebaseApply
		case pathExists(".git/rebase-apply/applying"):
			state.State = Am
		default:
			state.State = AmRebase
		}
	case pathExists(".git/MERGE_HEAD"):
		state.State = Merging
	case pathExists(".git/CHERRY_PICK_HEAD"):
		state.State = CherryPick
	case pathExists(".git/REVERT_HEAD"):
		state.State = Reverting
	case pathExists(".git/BISECT_LOG"):
		state.State = Bisecting
	}

	return state, nil
}

// pathExists checks if a file or directory exists.
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

// readInt reads an integer from a file.
func readInt(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read file: %w", err)
	}

	i, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0, fmt.Errorf("parse int: %w", err)
	}

	return i, nil
}

// gitStatus retrieves the Git repository status.
func gitStatus(path string) (string, error) {
	stdout, err := exec.Command(
		"git",
		"-C",
		path,
		"status",
		"--porcelain=2",
		"--branch",
		"--show-stash",
	).Output()
	if err != nil {
		return "", fmt.Errorf("run cmd: %w", err)
	}

	return string(stdout), nil
}

// parseStatus parses the Git repository status output.
func parseStatus(output string) (*Status, error) {
	status := &Status{}

	for _, line := range strings.Split(output, "\n") {
		s := strings.Split(line, " ")
		switch s[0] {
		case "#":
			switch s[1] {
			case "branch.oid":
				status.Commit = s[2]
			case "branch.head":
				status.Branch = s[2]
			case "stash":
				numStashed, err := strconv.Atoi(s[2])
				if err != nil {
					return nil, fmt.Errorf("parse num stashed: %w", err)
				}
				status.Stashed = numStashed
			case "branch.upstream":
				status.Upstream = s[2]
			case "branch.ab":
				ahead, err := strconv.Atoi(s[2][1:])
				if err != nil {
					return nil, fmt.Errorf("parse ahead: %w", err)
				}
				status.Ahead = ahead

				behind, err := strconv.Atoi(s[3][1:])
				if err != nil {
					return nil, fmt.Errorf("parse behind: %w", err)
				}
				status.Behind = behind
			}
		case "1", "2":
			if slices.Contains([]string{"DD", "AU", "UD", "UA", "DU", "AA", "UU"}, s[1]) {
				status.Conflict++
			} else if s[1][1] == 'M' {
				status.Modified++
			} else {
				status.Staged++
			}
		case "?":
			status.Untracked++
		}
	}

	return status, nil
}

// buildOutput builds the final output string based on the Git repository status.
func buildOutput(status Status, state State, symbols Symbols) string {
	var b strings.Builder
	b.WriteString(symbols.Prefix)

	if status.Branch == "(detached)" {
		b.WriteString(fmt.Sprintf(":%s", status.Commit[:7]))
	} else {
		b.WriteString(status.Branch)

		if status.Upstream == "" {
			b.WriteString(fmt.Sprintf(" %s", symbols.Local))
		} else {
			b.WriteString(fmt.Sprintf(" {%s}", status.Upstream))
		}

		if status.Ahead > 0 || status.Behind > 0 {
			b.WriteString(" ")

			if status.Ahead > 0 {
				b.WriteString(fmt.Sprintf("%s%d", symbols.Ahead, status.Ahead))
			}

			if status.Behind > 0 {
				b.WriteString(fmt.Sprintf("%s%d", symbols.Behind, status.Behind))
			}
		}
	}

	b.WriteString(symbols.Sep)

	if state.State != "" {
		b.WriteString(state.State)

		if state.Total > 0 {
			b.WriteString(fmt.Sprintf(" %d/%d", state.Step, state.Total))
		}

		b.WriteString(symbols.Sep)
	}

	if status.Staged > 0 {
		b.WriteString(fmt.Sprintf("%s%d", symbols.Staged, status.Staged))
	}
	if status.Conflict > 0 {
		b.WriteString(fmt.Sprintf("%s%d", symbols.Conflict, status.Conflict))
	}
	if status.Modified > 0 {
		b.WriteString(fmt.Sprintf("%s%d", symbols.Modified, status.Modified))
	}
	if status.Untracked > 0 {
		b.WriteString(fmt.Sprintf("%s%d", symbols.Untracked, status.Untracked))
	}
	if status.Stashed > 0 {
		b.WriteString(fmt.Sprintf("%s%d", symbols.Stashed, status.Stashed))
	}

	if status.Staged == 0 && status.Conflict == 0 && status.Modified == 0 && status.Untracked == 0 && status.Stashed == 0 {
		b.WriteString(symbols.Clean)
	}

	b.WriteString(symbols.Suffix)

	return b.String()
}
