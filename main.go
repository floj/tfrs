package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/floj/tfrs/chooser"
	"github.com/floj/tfrs/chooser/fuzzy"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/mattn/go-isatty"
)

const allMarker = "<all>"

func getRessources(dir, prefix, modulesDir string, depth, maxDepth int) []string {
	if depth > maxDepth {
		return []string{}
	}

	module, diags := tfconfig.LoadModule(dir)
	if diags.HasErrors() {
		fmt.Fprintln(os.Stderr, diags)
	}

	modules := make([]string, 0, len(module.ModuleCalls))
	for _, v := range module.ModuleCalls {
		name := prefix + "module." + v.Name
		modules = append(modules, name)
		src := filepath.Join(dir, v.Source)
		if !strings.HasPrefix(v.Source, "./") || !strings.HasPrefix(v.Source, "../") {
			src = filepath.Join(modulesDir, strings.ReplaceAll(name, "module.", ""))
		}
		submodules := getRessources(src, name+".", modulesDir, depth+1, maxDepth)
		modules = append(modules, submodules...)
	}

	ressources := make([]string, 0, len(module.ManagedResources))
	for _, v := range module.ManagedResources {
		ressources = append(ressources, prefix+v.Type+"."+v.Name)
	}

	sort.Strings(modules)
	sort.Strings(ressources)

	names := make([]string, 0, len(modules)+len(ressources))
	names = append(names, ressources...)
	names = append(names, modules...)
	return names
}

func main() {
	chdir := flag.String("chdir", ".", "lookup resources from this directory")
	maxDepth := flag.Int("depth", 0, "how many levels to decent into submodules")
	flag.Parse()

	if isatty.IsTerminal(os.Stdout.Fd()) && len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "If used in tty mode, at least one argument must be passed, eg: %s plan\n", os.Args[0])
		os.Exit(1)
	}

	resources := getRessources(*chdir, "", filepath.Join(*chdir, ".terraform/modules"), 0, *maxDepth)
	if len(resources) == 0 {
		fmt.Fprintf(os.Stderr, "No modules or resources found in %s\n", *chdir)
		os.Exit(1)
	}

	names := []string{allMarker}
	names = append(names, resources...)

	var chooser chooser.Chooser
	chooser = fuzzy.NewChooser()

	ok, selected, err := chooser.Choose(names)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	if !ok {
		os.Exit(1)
	}

	for i := 0; i < len(selected); i++ {
		if selected[i] == allMarker {
			selected = []string{}
			break
		}
		selected[i] = "-target=" + selected[i]
	}
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Println(strings.Join(selected, " "))
		return
	}

	var args []string
	args = append(args, "terraform")
	args = append(args, flag.Args()...)
	args = append(args, selected...)

	fmt.Printf("> %s\n", strings.Join(args, " "))
	bin, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not find %s in PATH: %v", args[0], err)
		os.Exit(1)
	}
	err = syscall.Exec(bin, args, os.Environ())
	fmt.Fprintf(os.Stderr, "%v", err)
	os.Exit(1)
}
