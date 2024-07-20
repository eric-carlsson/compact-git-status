package main

import (
	"fmt"
	"log"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

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

func main() {
	symbols := &Symbols{
		Prefix:    "[",
		Suffix:    "]",
		Sep:       "|",
		Local:     "L",
		Modified:  "✚ ",
		Staged:    "● ",
		Conflict:  "✖ ",
		Untracked: "…",
		Stashed:   "⚑ ",
		Ahead:     "↑·",
		Behind:    "↓·",
		Clean:     "✔",
	}

	output, err := gitStatus()
	if err != nil {
		log.Fatal(err)
	}

	status, err := parseStatus(output)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(buildOutput(status, symbols))
}

func gitStatus() (string, error) {
	cmd := exec.Command(
		"git",
		"status",
		"--porcelain=2",
		"--branch",
		"--show-stash",
	)

	stdout, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("run cmd: %w", err)
	}

	return string(stdout), nil
}

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

func buildOutput(status *Status, symbols *Symbols) string {
	var b strings.Builder
	b.WriteString(symbols.Prefix)

	if status.Branch == "(detached)" {
		b.WriteString(fmt.Sprintf(":%s", status.Commit[:7]))
	} else {
		b.WriteString(status.Branch)
	}

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

	b.WriteString(symbols.Sep)

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

	b.WriteString(symbols.Suffix)

	return b.String()
}
