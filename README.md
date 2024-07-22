# compact-git-status

A utility for printing compact information about the current git repository.

Inspired by [bash-git-prompt](https://github.com/magicmonty/bash-git-prompt/), this is a portable reimplementaton that has no external dependencies (except `git`). The program is used as a command line tool and prints to stdout. You are free to call it however and from wherever you want.

## Using in tmux

You can call this utility from tmux to print information about the active pane's git repository to the status bar. For example tmux configuration see below.

```shell
set -g status-right "#( compact-git-status --path #{pane_current_path} )"
```
