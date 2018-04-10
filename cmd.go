package cmd

import (
	"errors"

	"github.com/beevik/prefixtree"
)

// A Command represents either a single named command or a new subtree of
// commands.
type Command struct {
	Name        string      // command string
	Shortcut    string      // optional shortcut for command
	Description string      // short description shown in command list help
	HelpText    string      // help text displayed for the command
	Alias       string      // command's alias (usually nested)
	Param       interface{} // user-defined parameter for this command
	Tree        *Tree       // the tree this command belongs to
	Subcommands *Tree       // the command's subtree of commands
}

// A Tree contains one or more commands which are grouped together and
// may be looked up by a shortest unambiguous prefix match.
type Tree struct {
	Title    string    // Description of all commands in tree
	Commands []Command // All commands in the tree
	tree     *prefixtree.Tree
}

// A Selection represents the result of looking up a command in a
// hierarchical command tree. It includes the whitespace-delimited arguments
// following the discovered command, if any.
type Selection struct {
	Command *Command // The selected command
	Args    []string // the command's white-space delimited arguments
}

// Errors returned by the cmd package.
var (
	ErrAmbiguous = errors.New("Command is ambiguous")
	ErrNotFound  = errors.New("Command not found")
)

// NewTree creates a new command tree containing all of the listed
// commands.
func NewTree(title string, commands []Command) *Tree {
	c := &Tree{
		Title:    title,
		Commands: commands,
		tree:     prefixtree.New(),
	}
	for i, cc := range c.Commands {
		c.Commands[i].Tree = c
		c.tree.Add(cc.Name, &c.Commands[i])
		if cc.Shortcut != "" {
			c.tree.Add(cc.Shortcut, &c.Commands[i])
		}
	}
	return c
}

// Lookup performs a hierarchical search on a command tree for a matching
// command.
func (c *Tree) Lookup(line string) (Selection, error) {
	cmdStr, argStr := split2(line)

	if cmdStr == "" {
		return Selection{}, nil
	}

	ci, err := c.tree.Find(cmdStr)
	switch err {
	case prefixtree.ErrPrefixAmbiguous:
		return Selection{}, ErrAmbiguous
	case prefixtree.ErrPrefixNotFound:
		return Selection{}, ErrNotFound
	}

	cmd := ci.(*Command)

	if cmd.Alias != "" {
		line = cmd.Alias + " " + argStr
		return c.Lookup(line)
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
