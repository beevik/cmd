package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func buildTree1() *Tree {
	tree := NewTree(TreeDescriptor{Name: "tree"})
	tree.AddCommand(CommandDescriptor{Name: "quit", Brief: "quit the application", Description: "Quit the application.", Usage: "quit", Data: "quit"})
	tree.AddCommand(CommandDescriptor{Name: "verylongstring", Brief: "very long string", Description: "This is a very long string.", Data: "verylongstring"})

	file := tree.AddSubtree(TreeDescriptor{Name: "file", Brief: "file commands", Description: "Commands that operate on files."})
	file.AddCommand(CommandDescriptor{Name: "open", Brief: "open a file", Description: "Open the specified file.", Usage: "file open <filename>", Data: "open"})
	file.AddCommand(CommandDescriptor{Name: "close", Brief: "close a file", Description: "Close the specified file.", Data: "close"})
	file.AddCommand(CommandDescriptor{Name: "read", Brief: "read a file", Description: "Read the specified file.", Data: "read"})
	file.AddCommand(CommandDescriptor{Name: "write", Brief: "write a file", Data: "write"})
	file.AddCommand(CommandDescriptor{Name: "run", Data: "run"})

	tree.AddShortcut("f", "file open")
	tree.AddShortcut("zz", "file open")
	tree.AddShortcut("xx", "file open")
	tree.AddShortcut("yy", "file open")
	tree.AddShortcut("dd", "file open")

	return tree
}

func buildTree2() *Tree {
	json := `{
	"tree": {
		"cmds": [
			{
				"name": "quit",
				"brief": "quit the application",
				"description": "Quit the application.",
				"usage": "quit"
			},
			{
				"name": "verylongstring",
				"brief": "very long string",
				"description": "This is a very long string."
			},
			{
				"name": "file",
				"brief": "file commands",
				"description": "Commands that operate on files.",
				"cmds": [
					{
						"name": "open",
						"brief": "open a file",
						"description": "Open the specified file.",
						"usage": "file open <filename>",
						"shortcuts": ["f", "zz", "xx", "yy", "dd" ]
					},
					{
						"name": "close",
						"brief": "close a file",
						"description": "Close the specified file."
					},
					{
						"name": "read",
						"brief": "read a file",
						"description": "Read the specified file."
					},
					{
						"name": "write",
						"brief": "write a file"
					},
					{
						"name": "run"
					}
				]
			}
		]
	}
}`

	tree, err := NewTreeFromJSON(json)
	if err != nil {
		panic(err)
	}

	tree.SetData("quit", "quit")
	tree.SetData("verylongstring", "verylongstring")
	tree.SetData("file open", "open")
	tree.SetData("file close", "close")
	tree.SetData("file read", "read")
	tree.SetData("file write", "write")
	tree.SetData("file run", "run")

	return tree
}

func TestParent(t *testing.T) {
	tree := NewTree(TreeDescriptor{Name: "tree"})
	tree.AddCommand(CommandDescriptor{Name: "quit"})

	file := tree.AddSubtree(TreeDescriptor{Name: "file"})
	open := file.AddSubtree(TreeDescriptor{Name: "open"})
	file.AddCommand(CommandDescriptor{Name: "close"})
	file.AddCommand(CommandDescriptor{Name: "read"})

	open.AddCommand(CommandDescriptor{Name: "now"})
	open.AddCommand(CommandDescriptor{Name: "later"})

	tree.AddShortcut("f", "file open now")
	tree.AddShortcut("xx", "file close")
	tree.AddShortcut("yy", "file read")

	cases := []struct {
		line   string
		parent *Tree
	}{
		{"quit", tree},
		{"file", tree},
		{"file open", file},
		{"file open now", open},
		{"file open later", open},
		{"file close", file},
		{"file read", file},
		{"f", open},
		{"xx", file},
		{"yy", file},
	}
	for i, c := range cases {
		n, _, err := tree.Lookup(c.line)
		if err != nil {
			t.Errorf("Case %d: unexpected parent for '%s'.\nError: %v\n", i, c.line, err)
			continue
		}
		if n.Parent() != c.parent {
			t.Errorf("Case %d: unexpected result.\nGot:\n%v\nWanted:\n%v\n", i, n.Parent(), c.parent)
		}
	}
}

func TestLookup(t *testing.T) {
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
		{" 	fi open", "open", []string{}, ""},
		{"f o xyz", "open", []string{"o", "xyz"}, ""},
		{"fi o xyz", "open", []string{"xyz"}, ""},
		{"quit", "quit", []string{}, ""},
		{"q", "quit", []string{}, ""},
		{"dd 1  2  3 4", "open", []string{"1", "2", "3", "4"}, ""},
		{"d 1  2  3 4", "open", []string{"1", "2", "3", "4"}, ""},
		{"xx 1  2  3 4", "open", []string{"1", "2", "3", "4"}, ""},
		{"x 1  2  3 4", "open", []string{"1", "2", "3", "4"}, ""},
	}

	run := func(t *testing.T, tree *Tree) {
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

	run(t, buildTree1())
	run(t, buildTree2())
}

func TestAutocomplete(t *testing.T) {
	// root
	//  alice [shortcut -> root.child.grandchild.alice]
	//  chair
	//	chairlift
	//  child
	//   sally
	//   steve
	//   grandchild
	//    alice
	//    mike
	tree := NewTree(TreeDescriptor{Name: "root"})
	tree.AddCommand(CommandDescriptor{Name: "chair"})
	tree.AddCommand(CommandDescriptor{Name: "chairlift"})

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
		{"", []string{"alice", "chair", "chairlift", "child"}},
		{"x", []string{}},
		{"a", []string{"alice"}},
		{"al", []string{"alice"}},
		{"alice", []string{"alice"}},
		{"c", []string{"chair", "chairlift", "child"}},
		{"ch", []string{"chair", "chairlift", "child"}},
		{"cha", []string{"chair", "chairlift"}},
		{"chai", []string{"chair", "chairlift"}},
		{"chair", []string{"chair"}},
		{"chairl", []string{"chairlift"}},
		{"chairli", []string{"chairlift"}},
		{"chairlif", []string{"chairlift"}},
		{"chairlift", []string{"chairlift"}},
		{"chairlift foo", []string{}},
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
				"    write  write a file\n" +
				"\n",
		},
		{
			"file open",
			"Usage: file open <filename>\n" +
				"Description:\n" +
				"   Open the specified file.\n" +
				"\n" +
				"Shortcuts: dd, f, xx, yy, zz\n" +
				"\n",
		},
		{
			"file run",
			"",
		},
		{
			"file read",
			"Description:\n" +
				"   Read the specified file.\n" +
				"\n",
		},
		{
			"xx",
			"Usage: file open <filename>\n" +
				"Description:\n" +
				"   Open the specified file.\n" +
				"\n" +
				"Shortcuts: dd, f, xx, yy, zz\n" +
				"\n",
		},
		{
			"quit",
			"Usage: quit\n" +
				"Description:\n" +
				"   Quit the application.\n" +
				"\n",
		},
	}

	run := func(t *testing.T, tree *Tree) {
		for _, c := range cases {
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

	run(t, buildTree1())
	run(t, buildTree2())
}
