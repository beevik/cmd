package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func buildTree() *Tree {
	tree := NewTree(TreeDescriptor{Name: "tree"})
	tree.AddCommand(CommandDescriptor{Name: "quit", Brief: "quit the application", Data: "quit"})
	tree.AddCommand(CommandDescriptor{Name: "verylongstring", Brief: "very long string"})

	file := tree.AddSubtree(TreeDescriptor{Name: "file", Brief: "file commands"})
	file.AddCommand(CommandDescriptor{Name: "open", Brief: "open a file", Data: "open"})
	file.AddCommand(CommandDescriptor{Name: "close", Brief: "close a file", Data: "close"})
	file.AddCommand(CommandDescriptor{Name: "read", Description: "read file description.", Brief: "read a file", Data: "read"})
	file.AddCommand(CommandDescriptor{Name: "write", Data: "write"})
	file.AddCommand(CommandDescriptor{Name: "run", Data: "run"})

	tree.AddShortcut("zz", "file open")
	tree.AddShortcut("xx", "file open")
	tree.AddShortcut("yy", "file open")
	tree.AddShortcut("dd", "file open")

	return tree
}

func TestLookup(t *testing.T) {
	tree := buildTree()

	cases := []struct {
		line string
		data string
		args []string
		err  string
	}{
		{"", "", nil, "Command not found"},
		{"foo", "", nil, "Command not found"},
		{"xyz abc", "", nil, "Command not found"},
		{"file r", "", nil, "Command is ambiguous"},
		{"fi ro", "", nil, "Command not found"},
		{"file x", "", nil, "Command not found"},
		{"file open foo 12", "open", []string{"foo", "12"}, ""},
		{"file	open	foo   12  ", "open", []string{"foo", "12"}, ""},
		{"\"file\"	open	foo   12  ", "open", []string{"foo", "12"}, ""},
		{" 	file	open	foo   12  ", "open", []string{"foo", "12"}, ""},
		{" 	file	open	\"foo\"   \"12\"  ", "open", []string{"foo", "12"}, ""},
		{" 	file	open	\"foo   12\"  ", "open", []string{"foo   12"}, ""},
		{" 	file	\"open\"	\"foo   12\"  ", "open", []string{"foo   12"}, ""},
		{" 	f open", "open", []string{}, ""},
		{"f o xyz", "open", []string{"xyz"}, ""},
		{"quit", "quit", []string{}, ""},
		{"q", "quit", []string{}, ""},
		{"dd 1  2  3 4", "open", []string{"1", "2", "3", "4"}, ""},
		{"d 1  2  3 4", "open", []string{"1", "2", "3", "4"}, ""},
		{"xx 1  2  3 4", "open", []string{"1", "2", "3", "4"}, ""},
		{"x 1  2  3 4", "open", []string{"1", "2", "3", "4"}, ""},
	}

	for i, c := range cases {
		n, args, err := tree.Lookup(c.line)
		cmd, _ := n.(*Command)
		argMismatch := false
		switch {
		case err == nil && c.err != "":
			t.Errorf("Case %d: Expected error '%s', but got no error\n", i, c.err)
		case err != nil && c.err == "":
			t.Errorf("Case %d: unexpected error '%v'", i, err)
		case err != nil && c.err != err.Error():
			t.Errorf("Case %d: expected error '%s', got '%s'.\n", i, c.err, err.Error())
		case err != nil && c.err == err.Error():
			continue
		case cmd != nil && cmd.Data != c.data:
			t.Errorf("Case %d: expected param '%s', got '%s'\n", i, c.data, cmd.Data)
		case len(args) != len(c.args):
			argMismatch = true
		default:
			for j := 0; j < len(args); j++ {
				if args[j] != c.args[j] {
					argMismatch = true
				}
			}
		}
		if argMismatch {
			t.Errorf("Case %d: args mismatch.\nEXPECTED: [%s]\nGOT: [%s]\n",
				i, strings.Join(c.args, ", "), strings.Join(args, ", "))
		}
	}
}

func TestAutocomplete(t *testing.T) {
	// root
	//  alice [shortcut -> root.child.grandchild.alice]
	//  chair
	//  child
	//   sally
	//   steve
	//   grandchild
	//    alice
	//    mike
	tree := NewTree(TreeDescriptor{Name: "root"})
	tree.AddCommand(CommandDescriptor{Name: "chair"})

	child := tree.AddSubtree(TreeDescriptor{Name: "child"})
	child.AddCommand(CommandDescriptor{Name: "sally"})
	child.AddCommand(CommandDescriptor{Name: "steve"})

	grandchild := child.AddSubtree(TreeDescriptor{Name: "grandchild"})
	grandchild.AddCommand(CommandDescriptor{Name: "alice"})
	grandchild.AddCommand(CommandDescriptor{Name: "mike"})

	tree.AddShortcut("alice", "child grandchild alice")

	cases := []struct {
		line    string
		matches []string
	}{
		{"", []string{"alice", "chair", "child"}},
		{"x", []string{}},
		{"a", []string{"alice"}},
		{"al", []string{"alice"}},
		{"alice", []string{"alice"}},
		{"c", []string{"chair", "child"}},
		{"ch", []string{"chair", "child"}},
		{"cha", []string{"chair"}},
		{"chai", []string{"chair"}},
		{"chair", []string{"chair"}},
		{"chair foo", []string{}},
		{"chairs", []string{}},
		{"chi", []string{"child"}},
		{"chil", []string{"child"}},
		{"child", []string{"child grandchild", "child sally", "child steve"}},
		{"childfoo", []string{}},
		{"child foo", []string{}},
		{"child s", []string{"child sally", "child steve"}},
		{"ch s", []string{}},
		{"chi s", []string{"child sally", "child steve"}},
		{"child sa", []string{"child sally"}},
		{"chi sa", []string{"child sally"}},
		{"child sally", []string{"child sally"}},
		{"child sally foo", []string{}},
		{"child st", []string{"child steve"}},
		{"child steve", []string{"child steve"}},
		{"child steve foo", []string{}},
		{"child g", []string{"child grandchild"}},
		{"c g", []string{}},
		{"ch g", []string{}},
		{"cha g", []string{}},
		{"chi g", []string{"child grandchild"}},
		{"chil g", []string{"child grandchild"}},
		{"child g", []string{"child grandchild"}},
		{"child gr", []string{"child grandchild"}},
		{"child grandchild", []string{"child grandchild alice", "child grandchild mike"}},
		{"chi grandchild", []string{"child grandchild alice", "child grandchild mike"}},
		{"ch grandchild", []string{}},
		{"child gr foo", []string{}},
		{"child grandchild foo", []string{}},
		{"child grandchild a", []string{"child grandchild alice"}},
		{"child grandchild alice", []string{"child grandchild alice"}},
		{"child grandchild m", []string{"child grandchild mike"}},
		{"child grandchild mike", []string{"child grandchild mike"}},
		{"child grandchild mike foo", []string{}},
		{"chi gr m", []string{"child grandchild mike"}},
		{"chi gr a", []string{"child grandchild alice"}},
		{"chi gr al", []string{"child grandchild alice"}},
		{"chi g alice", []string{"child grandchild alice"}},
		{"ch grandchild alice", []string{}},
	}

	for i, c := range cases {
		matches := tree.Autocomplete(c.line)
		mismatch := false
		if len(matches) != len(c.matches) {
			mismatch = true
		} else {
			for j := 0; j < len(matches); j++ {
				if matches[j] != c.matches[j] {
					mismatch = true
					break
				}
			}
		}
		if mismatch {
			t.Errorf("Case %d: Result mismatch.\nEXPECTED: [%s]\nGOT: [%s]\n",
				i, strings.Join(c.matches, ", "), strings.Join(matches, ", "))
		}
	}
}

func TestGetHelp(t *testing.T) {
	cases := []struct {
		line string
		help string
	}{
		{
			"",
			"tree commands:\n" +
				"    file            file commands\n" +
				"    quit            quit the application\n" +
				"    verylongstring  very long string\n" +
				"\n",
		},
		{
			"file",
			"file commands:\n" +
				"    close  close a file\n" +
				"    open   open a file\n" +
				"    read   read a file\n" +
				"\n",
		},
		{
			"file open",
			"Description:\n" +
				"   open a file.\n" +
				"\n" +
				"Shortcuts: dd, xx, yy, zz\n" +
				"\n",
		},
		{
			"file run",
			"",
		},
		{
			"file read",
			"Description:\n" +
				"   read file description.\n" +
				"\n",
		},
		{
			"xx",
			"Description:\n" +
				"   open a file.\n" +
				"\n" +
				"Shortcuts: dd, xx, yy, zz\n" +
				"\n",
		},
		{
			"quit",
			"Description:\n" +
				"   quit the application.\n" +
				"\n",
		},
	}

	for _, c := range cases {
		tree := buildTree()
		buf := new(bytes.Buffer)
		tree.GetHelp(buf, strings.Fields(c.line))
		help := buf.String()
		if help != c.help {
			t.Errorf("DisplayCommands produced unexpected result.\n"+
				"EXPECTED:\n%s\nGOT:\n%s\n",
				c.help, help)
		}
	}
}
