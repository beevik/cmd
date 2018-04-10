package cmd

import (
	"errors"

	"github.com/beevik/prefixtree"
)

// A Tree contains one or more commands which are grouped together and may be
// looked up by a shortest unambiguous prefix match.
type Tree struct {
	Title    string     // description of all commands in tree
	Commands []*Command // all commands in the tree
	pt       *prefixtree.Tree
}

// A Command represents either a single named command or a new subtree of
// commands.
type Command struct {
	Name        string      // command string
	Shortcut    string      // optional shortcut for command
	Brief       string      // brief description shown in command list
	Description string      // short description shown in command list help
	HelpText    string      // help text displayed for the command
	Alias       string      // command's alias (usually nested)
	Subcommands *Tree       // the command's subtree of commands
	Data        interface{} // user-defined data for this command
}

// A Selection represents the result of looking up a command in a command
// tree. It includes the whitespace-delimited arguments following the
// discovered command.
type Selection struct {
	Command *Command // the selected command
	Args    []string // the command's white-space delimited arguments
}

// Errors returned by the cmd package.
var (
	ErrAmbiguous = errors.New("Command is ambiguous")
	ErrNotFound  = errors.New("Command not found")
)

// NewTree creates a new command tree containing all of the listed commands.
func NewTree(title string, commands []Command) *Tree {
	t := &Tree{
		Title:    title,
		Commands: make([]*Command, len(commands)),
		pt:       prefixtree.New(),
	}

	for i, c := range commands {
		t.Commands[i] = new(Command)
		*t.Commands[i] = c
		t.pt.Add(c.Name, t.Commands[i])
		if c.Shortcut != "" {
			t.pt.Add(c.Shortcut, t.Commands[i])
		}
	}
	return t
}

// Lookup performs a search on a command tree for a matching command.
func (t *Tree) Lookup(line string) (Selection, error) {
	cmdStr, argStr := split2(line)

	if cmdStr == "" {
		return Selection{}, nil
	}

	ci, err := t.pt.Find(cmdStr)
	switch err {
	case prefixtree.ErrPrefixAmbiguous:
		return Selection{}, ErrAmbiguous
	case prefixtree.ErrPrefixNotFound:
		return Selection{}, ErrNotFound
	}

	cmd := ci.(*Command)

	if cmd.Alias != "" {
		line = cmd.Alias + " " + argStr
		return t.Lookup(line)
	}

	if cmd.Subcommands != nil && argStr != "" {
		return cmd.Subcommands.Lookup(argStr)
	}

	args := splitArgs(argStr)
	return Selection{Command: cmd, Args: args}, nil
}

func split2(s string) (cmd, args string) {
	return nextToken(stripLeadingWhitespace(s))
}

func splitArgs(args string) []string {
	args = stripLeadingWhitespace(args)

	ss := make([]string, 0)
	for len(args) > 0 {
		var arg string
		arg, args = nextToken(args)
		ss = append(ss, arg)
	}

	if len(args) > 0 {
		ss = append(ss, args)
	}
	return ss
}

func nextToken(s string) (token, remain string) {
	for i, c := range s {
		if c == ' ' || c == '\t' {
			return s[:i], stripLeadingWhitespace(s[i:])
		}
	}
	return s, ""
}

func stripLeadingWhitespace(s string) string {
	for i, c := range s {
		if c != ' ' && c != '\t' {
			return s[i:]
		}
	}
	return ""
}
