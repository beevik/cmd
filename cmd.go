package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/beevik/prefixtree"
)

// A Tree contains one or more commands which are grouped together and may be
// looked up by a shortest unambiguous prefix match.
type Tree struct {
	Title    string     // description of all commands in tree
	Commands []*Command // all commands in the tree
	pt       *prefixtree.Tree
}

// A Command represents either a single named command or the root of a subtree
// of commands.
type Command struct {
	Name        string      // command string
	Brief       string      // brief description shown in a command list
	Description string      // long description shown with command help
	Usage       string      // usage hint text
	Shortcuts   []string    // command shortcuts
	Subtree     *Tree       // the command's subtree of commands
	Data        interface{} // user-defined data for this command
}

// DisplayUsage outputs the command's usage string.
func (c *Command) DisplayUsage(w io.Writer) {
	if c.Usage != "" {
		fmt.Fprintf(w, "Usage: %s\n", c.Usage)
	}
}

// DisplayShortcuts displays all shortcuts associated with the command.
func (c *Command) DisplayShortcuts(w io.Writer) {
	if c.Shortcuts != nil {
		switch {
		case len(c.Shortcuts) > 1:
			fmt.Fprintf(w, "Shortcuts: %s\n\n", strings.Join(c.Shortcuts, ", "))
		default:
			fmt.Fprintf(w, "Shortcut: %s\n\n", c.Shortcuts[0])
		}
	}
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

// NewTree creates a new command tree with the given title.
func NewTree(title string) *Tree {
	return &Tree{
		Title:    title,
		Commands: make([]*Command, 0),
		pt:       prefixtree.New(),
	}
}

// AddCommand adds a command to a command tree.
func (t *Tree) AddCommand(c Command) *Command {
	cc := &Command{}
	*cc = c
	t.Commands = append(t.Commands, cc)
	t.pt.Add(c.Name, cc)
	return cc
}

// AddShortcut adds a shortcut to a command in the tree.
func (t *Tree) AddShortcut(shortcut, target string) error {
	if len(strings.Fields(shortcut)) > 1 {
		return errors.New("invalid shortcut")
	}

	cmd, _, err := t.lookupCommand(target)
	if err != nil {
		return err
	}

	cmd.Shortcuts = append(cmd.Shortcuts, shortcut)
	t.pt.Add(shortcut, cmd)
	return nil
}

// DisplayHelp parses the 'help' command's arguments string and displays
// an appropriate help response.
func (t *Tree) DisplayHelp(w io.Writer, args []string) error {
	switch {
	case len(args) == 0:
		t.DisplayCommands(w)
	default:
		s, err := t.Lookup(strings.Join(args, " "))
		switch {
		case err != nil:
			return err
		case s.Command.Subtree != nil:
			s.Command.Subtree.DisplayCommands(w)
		default:
			s.Command.DisplayUsage(w)
			switch {
			case s.Command.Description != "":
				fmt.Fprintf(w, "Description:\n%s\n\n", indentWrap(3, s.Command.Description))
			case s.Command.Brief != "":
				fmt.Fprintf(w, "Description:\n%s.\n\n", indentWrap(3, s.Command.Brief))
			}
		}
		s.Command.DisplayShortcuts(w)
	}
	return nil
}

func indentWrap(indent int, s string) string {
	ss := strings.Fields(s)
	if len(ss) == 0 {
		return ""
	}

	counts := make([]int, 0)
	count := 1
	l := indent + len(ss[0])
	for i := 1; i < len(ss); i++ {
		if l+1+len(ss[i]) < 80 {
			count++
			l += 1 + len(ss[i])
			continue
		}

		counts = append(counts, count)
		count = 1
		l = indent + len(ss[i])
	}
	counts = append(counts, count)

	var lines []string
	i := 0
	for _, c := range counts {
		line := strings.Repeat(" ", indent) + strings.Join(ss[i:i+c], " ")
		lines = append(lines, line)
		i += c
	}

	return strings.Join(lines, "\n")
}

// DisplayCommands displays the commands available at the tree's top level.
func (t *Tree) DisplayCommands(w io.Writer) {
	maxNameLen := 0
	for _, c := range t.Commands {
		if len(c.Name) > maxNameLen {
			maxNameLen = len(c.Name)
		}
	}

	fmt.Fprintf(w, "%s commands:\n", t.Title)
	for _, c := range t.Commands {
		if c.Brief != "" {
			fmt.Fprintf(w, "    %-*s  %s\n", maxNameLen, c.Name, c.Brief)
		}
	}
	fmt.Fprintln(w)
}

// Lookup performs a search on a command tree for a matching command. If
// found, it returns the command and the command arguments.
func (t *Tree) Lookup(line string) (Selection, error) {
	cmd, args, err := t.lookupCommand(line)
	if err != nil {
		return Selection{}, err
	}

	return Selection{cmd, args}, nil
}

func (t *Tree) lookupCommand(line string) (cmd *Command, args []string, err error) {
	cmdStr, argStr := split2(line)

	args = make([]string, 0)
	if cmdStr == "" {
		return cmd, args, nil
	}

	pt := t.pt
	for {
		ci, err := pt.Find(cmdStr)
		switch err {
		case prefixtree.ErrPrefixAmbiguous:
			return cmd, args, ErrAmbiguous
		case prefixtree.ErrPrefixNotFound:
			return cmd, args, ErrNotFound
		}

		cmd = ci.(*Command)

		if cmd.Subtree == nil || argStr == "" {
			break
		}

		cmdStr, argStr = split2(argStr)
		pt = cmd.Subtree.pt
	}

	args = []string{}
	for argStr != "" {
		var arg string
		arg, argStr = nextField(argStr)
		args = append(args, arg)
	}
	return cmd, args, nil
}

func split2(s string) (cmd, args string) {
	return nextField(stripLeadingWhitespace(s))
}

func nextField(s string) (field, remain string) {
	if len(s) > 0 && s[0] == '"' {
		for i, c := range s[1:] {
			if c == '"' {
				return s[1 : i+1], stripLeadingWhitespace(s[i+2:])
			}
		}
		return s[1:], ""
	}

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
