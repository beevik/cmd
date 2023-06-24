package cmd

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/beevik/prefixtree/v2"
)

// A Node may be a Tree or a Command.
type Node interface {
	DisplayHelp(w io.Writer)
	name() string
	brief() string
}

// A TreeDescriptor describes a command tree.
type TreeDescriptor struct {
	Name        string // tree name
	Brief       string // brief description shown in a command list
	Description string // long description shown with command help
	Usage       string // usage hint text
	Data        any    // user-defined data
}

// A Tree contains one or more commands which are grouped together and may be
// looked up by a shortest unambiguous prefix match.
type Tree struct {
	TreeDescriptor
	commands []*Command
	subtrees []*Tree
	pt       *prefixtree.Tree[Node]
}

func (t *Tree) name() string {
	return t.Name
}

func (t *Tree) brief() string {
	return t.Brief
}

// Commands returns the tree's commands.
func (t *Tree) Commands() []*Command {
	return t.commands
}

// DisplayUsage outputs the tree's usage string.
func (t *Tree) DisplayUsage(w io.Writer) {
	if t.Usage != "" {
		fmt.Fprintf(w, "Usage: %s\n", t.Usage)
	} else {
		fmt.Fprintf(w, "Usage: %s [subcommand]\n", t.Name)
	}
}

// Subtrees returns the tree's subtrees.
func (t *Tree) Subtrees() []*Tree {
	return t.subtrees
}

// A CommandDescriptor describes a single command within a command tree.
type CommandDescriptor struct {
	Name        string // command name
	Brief       string // brief description shown in a command list
	Description string // long description shown with command help
	Usage       string // usage hint text
	Data        any    // user-defined data
}

// A Command represents either a single named command or the root of a subtree
// of commands.
type Command struct {
	CommandDescriptor
	shortcuts []string
}

func (c *Command) name() string {
	return c.Name
}

func (c *Command) brief() string {
	return c.Brief
}

// DisplayHelp outputs the help text associated with the command, including
// its usage, description, and shortcuts.
func (c *Command) DisplayHelp(w io.Writer) {
	c.DisplayUsage(w)
	c.DisplayDescription(w)
	c.DisplayShortcuts(w)
}

// DisplayUsage outputs the command's usage string.
func (c *Command) DisplayUsage(w io.Writer) {
	if c.Usage != "" {
		fmt.Fprintf(w, "Usage: %s\n", c.Usage)
	}
}

// DisplayDescription outputs the command's description text. If the
// command has no description, the commands 'brief' text is output instead.
func (c *Command) DisplayDescription(w io.Writer) {
	switch {
	case c.Description != "":
		fmt.Fprintf(w, "Description:\n%s\n\n", indentWrap(3, c.Description))
	case c.Brief != "":
		fmt.Fprintf(w, "Description:\n%s.\n\n", indentWrap(3, c.Brief))
	}
}

// DisplayShortcuts displays all shortcuts associated with the command.
func (c *Command) DisplayShortcuts(w io.Writer) {
	if c.shortcuts != nil {
		switch {
		case len(c.shortcuts) > 1:
			fmt.Fprintf(w, "Shortcuts: %s\n\n", strings.Join(c.shortcuts, ", "))
		default:
			fmt.Fprintf(w, "Shortcut: %s\n\n", c.shortcuts[0])
		}
	}
}

// Shortcuts returns the shortcut strings associated with the command.
func (c *Command) Shortcuts() []string {
	sort.Slice(c.shortcuts, func(i, j int) bool {
		return c.shortcuts[i] < c.shortcuts[j]
	})
	return c.shortcuts
}

// Errors returned by the cmd package.
var (
	ErrAmbiguous = errors.New("Command is ambiguous")
	ErrNotFound  = errors.New("Command not found")
)

// NewTree creates a new command tree with the given title.
func NewTree(d TreeDescriptor) *Tree {
	return &Tree{
		TreeDescriptor: d,
		commands:       nil,
		subtrees:       nil,
		pt:             prefixtree.New[Node](),
	}
}

// AddCommand adds a command to a command tree.
func (t *Tree) AddCommand(d CommandDescriptor) *Command {
	c := &Command{
		CommandDescriptor: d,
		shortcuts:         nil,
	}
	t.commands = append(t.commands, c)
	t.pt.Add(c.Name, c)
	return c
}

// AddShortcut adds a shortcut to a command in the tree.
func (t *Tree) AddShortcut(shortcut, target string) error {
	if len(strings.Fields(shortcut)) != 1 {
		return errors.New("invalid shortcut")
	}

	cmd, _, err := t.LookupCommand(target)
	if err != nil {
		return err
	}

	// Insert shortcut in alphabetical order
	i := sort.SearchStrings(cmd.shortcuts, shortcut)
	cmd.shortcuts = append(cmd.shortcuts, "")
	copy(cmd.shortcuts[i+1:], cmd.shortcuts[i:])
	cmd.shortcuts[i] = shortcut

	t.pt.Add(shortcut, cmd)
	return nil
}

// AddSubtree adds a child command tree to an existing command tree.
func (t *Tree) AddSubtree(d TreeDescriptor) *Tree {
	subtree := &Tree{
		TreeDescriptor: d,
		commands:       nil,
		subtrees:       nil,
		pt:             prefixtree.New[Node](),
	}
	t.subtrees = append(t.subtrees, subtree)
	t.pt.Add(subtree.Name, subtree)
	return subtree
}

// GetHelp parses the 'help' command's arguments string and displays
// an appropriate help response.
func (t *Tree) GetHelp(w io.Writer, args []string) error {
	var n Node
	switch {
	case len(args) == 0:
		n = t
	default:
		var err error
		n, _, err = t.Lookup(strings.Join(args, " "))
		if err != nil {
			return err
		}
	}

	n.DisplayHelp(w)
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

// DisplayHelp displays a sorted list of commands (and subtrees) available at
// the tree's top level.
func (t *Tree) DisplayHelp(w io.Writer) {
	nodes := make([]Node, 0)
	for _, c := range t.commands {
		nodes = append(nodes, c)
	}
	for _, st := range t.subtrees {
		nodes = append(nodes, st)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].name() < nodes[j].name()
	})

	maxNameLen := 0
	for _, e := range nodes {
		if len(e.name()) > maxNameLen {
			maxNameLen = len(e.name())
		}
	}

	fmt.Fprintf(w, "%s commands:\n", t.Name)
	for _, e := range nodes {
		if e.brief() != "" {
			fmt.Fprintf(w, "    %-*s  %s\n", maxNameLen, e.name(), e.brief())
		}
	}
	fmt.Fprintln(w)
}

// Autocomplete builds a list of auto-completion candidates for the provided
// line of text.
func (t *Tree) Autocomplete(line string) []string {
	field, remain := nextField(stripLeadingWhitespace(line))
	pt := t.pt
	prefix := ""
	for {
		matches := pt.FindKeyValues(field)
		if len(matches) == 0 {
			break
		}

		if len(matches) > 1 {
			if remain != "" {
				break
			}
			results := []string{}
			for _, match := range matches {
				results = append(results, prefix+match.Key)
			}
			return results
		}

		match := matches[0]
		if _, ok := match.Value.(*Command); ok {
			if remain != "" {
				break
			}
			return []string{prefix + match.Key}
		}

		subtree := match.Value.(*Tree)
		if remain == "" && field != subtree.Name {
			return []string{prefix + match.Key}
		}

		prefix += match.Key + " "
		pt = subtree.pt
		field, remain = nextField(remain)
	}

	return []string{}
}

// Lookup performs a search on a command tree for a command or subtree node
// matching the line input. If found, it returns the matching node and the
// remaining unmatched line arguments.
func (t *Tree) Lookup(line string) (n Node, args []string, err error) {
	field, remain := nextField(stripLeadingWhitespace(line))

	args = []string{}
	if field == "" {
		return nil, args, ErrNotFound
	}

	pt := t.pt
	for {
		v, err := pt.FindValue(field)
		switch err {
		case prefixtree.ErrPrefixAmbiguous:
			return nil, args, ErrAmbiguous
		case prefixtree.ErrPrefixNotFound:
			return nil, args, ErrNotFound
		}

		if _, ok := v.(*Command); ok {
			n = v
			break
		}

		subtree := v.(*Tree)
		if remain == "" {
			n = v
			break
		}

		field, remain = nextField(remain)
		pt = subtree.pt
	}

	for remain != "" {
		field, remain = nextField(remain)
		args = append(args, field)
	}
	return n, args, nil
}

// LookupCommand performs a search on a command tree for a command matching
// the line input. If found, it returns the matching command and the remaining
// unmatched line arguments.
func (t *Tree) LookupCommand(line string) (cmd *Command, args []string, err error) {
	var r any
	r, args, err = t.Lookup(line)
	if err != nil {
		return nil, nil, err
	}
	if cmd, ok := r.(*Command); ok {
		return cmd, args, nil
	}
	return nil, nil, ErrNotFound
}

// LookupSubtree performs a search on a command tree for a subtree matching
// the line input. If found, it returns the matching subtree and the remaining
// unmatched line arguments.
func (t *Tree) LookupSubtree(line string) (subtree *Tree, args []string, err error) {
	var r any
	r, args, err = t.Lookup(line)
	if err != nil {
		return nil, nil, err
	}
	if subtree, ok := r.(*Tree); ok {
		return subtree, args, nil
	}
	return nil, nil, ErrNotFound
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
