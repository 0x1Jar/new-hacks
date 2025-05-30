package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var modules = []module{
	&domains{},
}

func main() {
	flag.Parse()

	sc := bufio.NewScanner(os.Stdin)

	db, err := sql.Open("sqlite3", "./bbdb.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db: %s\n", err)
		return
	}

	if err = db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping db: %s\n", err)
		return
	}

	// initialise db file
	if flag.NArg() > 0 && flag.Arg(0) == "init" {
		err := initModules(db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "init error: %s\n", err)
		}
		return
	}

	// Handle commands from arguments
	if flag.NArg() >= 2 {
		action := flag.Arg(0)
		typ := flag.Arg(1)
		arg := ""
		if flag.NArg() > 2 {
			arg = flag.Arg(2)
		}

		// Construct the command line as if it came from stdin
		// This allows us to reuse the tokenize and processing logic
		cmdLine := fmt.Sprintf("%s %s %s", action, typ, arg)
		processCommand(strings.TrimSpace(cmdLine), db)
		return
	}

	// If no arguments are provided (or only 'init'), read from stdin
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		processCommand(line, db)
	}
}

func processCommand(line string, db *sql.DB) {
	op, err := tokenize(line)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %s\n", err)
		return
	}

	mod, err := getModule(op, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "module error: %s\n", err)
		return
	}

	switch op.action {
	case "add":
		err = mod.Add(op.arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "add error: %s\n", err)
			return
		}
		fmt.Printf("Added: %s\n", op.arg) // Provide feedback for CLI usage

	case "all":
		vals, err := mod.All()
		if err != nil {
			fmt.Fprintf(os.Stderr, "all error: %s\n", err)
			return
		}
		if len(vals) == 0 {
			fmt.Println("No domains found.") // Provide feedback for CLI usage
			return
		}
		for _, v := range vals {
			fmt.Println(v)
		}

	case "delete":
		err = mod.Delete(op.arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "delete error: %s\n", err) // Corrected "add error" to "delete error"
			return
		}
		fmt.Printf("Deleted: %s\n", op.arg) // Provide feedback for CLI usage
	default:
		fmt.Fprintf(os.Stderr, "unknown action: %s\n", op.action)
	}
}

type module interface {
	// meta
	names() []string
	setDB(*sql.DB)
	initModule() error

	// change
	Add(string) error
	Delete(string) error

	// read
	All() ([]string, error)
}

func getModule(o op, db *sql.DB) (module, error) {

	for _, m := range modules {
		for _, name := range m.names() {
			if name == o.typ {
				m.setDB(db)
				return m, nil
			}
		}
	}

	return nil, fmt.Errorf("no such module: %s", o.typ)
}

type op struct {
	action string
	typ    string
	arg    string
}

func tokenize(in string) (op, error) {
	t := strings.Fields(in)
	if len(t) < 2 {
		return op{}, fmt.Errorf("not enough tokens in '%s'", in)
	}

	arg := ""
	if len(t) > 2 {
		arg = t[2]
	}

	return op{
		action: strings.ToLower(t[0]),
		typ:    strings.ToLower(t[1]),
		arg:    arg,
	}, nil
}

func initModules(db *sql.DB) error {

	for _, m := range modules {
		m.setDB(db)
		err := m.initModule()
		if err != nil {
			return err
		}
	}

	return nil
}
