package cmd

import "testing"

func buildTree() *Tree {
	tree := NewTree("root", []Command{
		{Name: "quit", Data: "quit"},
		{Name: "file", Subcommands: NewTree("File", []Command{
			{Name: "open", Data: "open"},
			{Name: "close", Data: "close"},
			{Name: "read", Data: "read"},
			{Name: "write", Data: "write"},
			{Name: "run", Data: "run"},
		})},
	})
	return tree
}

func TestLookup(t *testing.T) {
	tree := buildTree()

	cases := []struct {
		s     string
		param string
		args  []string
		err   string
	}{
		{"xyz abc", "", nil, "Command not found"},
		{"fi ro", "", nil, "Command not found"},
		{"file r", "", nil, "Command is ambiguous"},
		{"file x", "", nil, "Command not found"},
		{" 	file	open	foo   12  ", "open", []string{"foo", "12"}, ""},
		{" 	f open", "open", []string{}, ""},
		{"f o xyz", "open", []string{"xyz"}, ""},
		{"quit", "quit", []string{}, ""},
		{"q", "quit", []string{}, ""},
	}

	for i, c := range cases {
		sel, err := tree.Lookup(c.s)
		switch {
		case err == nil && c.err != "":
			t.Errorf("Case %d: Expected error '%s', but got no error\n", i, c.err)
		case err != nil && c.err == "":
			t.Errorf("Case %d: unexpected error '%v'", i, err)
		case err != nil && c.err != err.Error():
			t.Errorf("Case %d: expected error '%s', got '%s'.\n", i, c.err, err.Error())
		case err != nil && c.err == err.Error():
			continue
		case sel.Command.Data != c.param:
			t.Errorf("Case %d: expected param '%s', got '%s'\n", i, c.param, sel.Command.Data)
		case len(sel.Args) != len(c.args):
			t.Errorf("Case %d: expected %d args, got %d.\n", i, len(c.args), len(sel.Args))
		default:
			for j := 0; j < len(sel.Args); j++ {
				if sel.Args[j] != c.args[j] {
					t.Errorf("Case %d: expected arg%d to be '%s', got '%s'.\n", i, j, c.args[j], sel.Args[j])
				}
			}
		}
	}
}
