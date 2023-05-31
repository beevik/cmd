[![Build Status](https://travis-ci.org/beevik/cmd.svg?branch=master)](https://travis-ci.org/beevik/cmd)
[![GoDoc](https://godoc.org/github.com/beevik/cmd?status.svg)](https://godoc.org/github.com/beevik/cmd)

cmd
===

The `cmd` package is a lightweight, hierarchical command processor. It's
handy when you have a number of text commands you wish to organize into
a hierarchy.

For example, suppose you have written an application that uses the following
command hierarchy:

* file
   * open
   * close
   * read
   * write
* status
* quit

With each of these commands you have associated a callback function that is
called with user-supplied arguments whenever the command is matched.

Now consider what would happen if the application user types the following
command into the application:

```
file open foo.txt rw
```

This command string would be fed into a command tree's `Lookup` function,
which would return the callback associated with the `file/open` command
as well as a slice of string arguments `[]string{"foo.txt", "rw"}`.

The `cmd` package supports shortest unambiguous prefix matches, so the
following command would return the same results:

```
f o foo.txt rw
```

### Code examples

This code shows how the command tree used in the example above might be
created:

```go
tree := cmd.NewTree("root")
file := cmd.NewTree("file")

tree.AddCommand(cmd.Command{Name: "file", Subtree: file})
tree.AddCommand(cmd.Command{Name: "status", Brief: "Show status", Data: (*app).onStatus})
tree.AddCommand(cmd.Command{Name: "quit", Brief: "Quit application", Data: (*app).onQuit})

file.AddCommand(cmd.Command{Name: "open", Brief: "Open file", Data: (*app).onOpen})
file.AddCommand(cmd.Command{Name: "close", Brief: "Close file", Data: (*app).onClose})
file.AddCommand(cmd.Command{Name: "read", Brief: "Read file", Data: (*app).onRead})
file.AddCommand(cmd.Command{Name: "write", Brief: "Write file", Data: (*app).onWrite})
```

And here is how you might query the command tree:

```go
func (a *app) processCommand(s string) error {
    sel, err := tree.Lookup(s)
    switch {
        case err == cmd.ErrAmbiguous:
            fmt.Printf("Command '%s' is ambiguous.\n", s)
            return err
        case err == cmd.ErrNotFound:
            fmt.Printf("Command '%s' not found.\n", s)
            return err
        default:
            handler := sel.Command.Param.(func(a *app, args []string) error)
            return handler(a, sel.Args)
    }
}
```
