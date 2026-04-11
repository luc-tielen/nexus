package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/luc/nexus/internal/projects"
	"github.com/luc/nexus/internal/secrets"
)

func runProjectCmd(ps *projects.Store, ss *secrets.Store, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: nexus project <add|delete|list> ...")
		os.Exit(1)
	}
	switch args[0] {
	case "add":
		runProjectAdd(ps, args[1:])
	case "delete":
		runProjectDelete(ps, ss, args[1:])
	case "list":
		runProjectList(ps)
	default:
		fmt.Fprintf(os.Stderr, "nexus: unknown project command %q\n", args[0])
		os.Exit(1)
	}
}

func runProjectAdd(ps *projects.Store, args []string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: nexus project add <name> <path>")
		os.Exit(1)
	}
	name, path := args[0], args[1]
	if err := ps.Add(name, path); err != nil {
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
	// Re-fetch to show the stored (tilde-expanded) path.
	p, _ := ps.Get(name)
	fmt.Printf("project %q added (path: %s)\n", p.Name, p.Path)
}

func runProjectDelete(ps *projects.Store, ss *secrets.Store, args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: nexus project delete <name>")
		os.Exit(1)
	}
	name := args[0]

	p, err := ps.Get(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}

	secretKeys := ss.KeysForProject(name)

	fmt.Printf("About to delete project %q (path: %s)\n", p.Name, p.Path)
	if len(secretKeys) > 0 {
		fmt.Println("The following secrets will also be deleted:")
		for _, k := range secretKeys {
			fmt.Printf("  %s\n", k)
		}
	}
	fmt.Print("This action is irreversible. Type the project name to confirm: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.TrimSpace(scanner.Text()) != name {
		fmt.Println("aborted")
		os.Exit(0)
	}

	for _, k := range secretKeys {
		if err := ss.Delete(k); err != nil {
			fmt.Fprintf(os.Stderr, "nexus: deleting secret %q: %v\n", k, err)
			os.Exit(1)
		}
	}
	if err := ps.Delete(name); err != nil {
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
	fmt.Printf("project %q deleted\n", name)
}

func runProjectList(ps *projects.Store) {
	list, err := ps.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "nexus:", err)
		os.Exit(1)
	}
	if len(list) == 0 {
		fmt.Println("no projects configured")
		return
	}
	for _, p := range list {
		fmt.Printf("%-20s %s\n", p.Name, p.Path)
	}
}
