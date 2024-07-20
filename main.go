package main

import (
	"fmt"
	"log"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

func main() {
	output, err := gitStatus()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)

	status, err := parseStatus(output)
	if err != nil {
		log.Fatal(err)
	}

	prefix := "["
	suffix := "]"
	separator := "|"

	var b strings.Builder
	b.WriteString(prefix)
	b.WriteString(status.Branch)

	if status.UpstreamBranch == "" {
		b.WriteString(" L")
	} else {
		b.WriteString(fmt.Sprintf(" {%s}", status.UpstreamBranch))
	}

	b.WriteString(separator)

	if len(status.Staged) > 0 {
		b.WriteString(fmt.Sprintf("● %d", len(status.Staged)))
	}

	if len(status.Modified) > 0 {
		b.WriteString(fmt.Sprintf("✚ %d", len(status.Modified)))
	}

	if len(status.Untracked) > 0 {
		b.WriteString(fmt.Sprintf("…%d", len(status.Untracked)))
	}

	if status.NumStashed > 0 {
		b.WriteString(fmt.Sprintf("⚑ %d", status.NumStashed))
	}

	if len(status.Staged)+len(status.Modified)+len(status.Untracked)+status.NumStashed == 0 {
		b.WriteString("✔")
	}

	b.WriteString(suffix)

	fmt.Println(b.String())
}

type Status struct {
	Branch          string
	UpstreamBranch  string
	UpstreamChanges string
	Modified        []string
	Untracked       []string
	Staged          []string
	NumStashed      int
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
			case "branch.head":
				status.Branch = s[2]
			case "stash":
				numStashed, err := strconv.Atoi(s[2])
				if err != nil {
					return nil, fmt.Errorf("parse num stashed: %w", err)
				}
				status.NumStashed = numStashed
			case "branch.upstream":
				status.UpstreamBranch = s[2]
			case "branch.ab":
				status.UpstreamChanges = s[2]
			}
		case "1":
			if slices.Contains([]byte{'M', 'A'}, s[1][0]) {
				status.Staged = append(status.Staged, s[2])
			}
			if s[1][1] == 'M' {
				status.Modified = append(status.Modified, s[2])
			}
		case "?":
			status.Untracked = append(status.Untracked, s[1])
		}
	}

	return status, nil
}
