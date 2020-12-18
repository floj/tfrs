package main

import (
	"encoding/json"
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

type ModulesManifest struct {
	BaseDir string
	Modules []struct {
		Key string
		Dir string
	}
}

func (mm *ModulesManifest) FindDir(key string) string {
	for _, m := range mm.Modules {
		if m.Key == key {
			return filepath.Join(mm.BaseDir, m.Dir)
		}
	}
	return ""
}

func loadManifest(rootDir string) (*ModulesManifest, error) {
	path := filepath.Join(rootDir, ".terraform/modules/modules.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Could not load manifest file '%s': %w", path, err)
	}
	defer f.Close()
	var m ModulesManifest
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, fmt.Errorf("Could not decode manifest file '%s': %w", path, err)
	}
	m.BaseDir = rootDir
	return &m, nil
}

func getRessources(dir, prefix string, mm *ModulesManifest, depth, maxDepth int) []string {
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

		key := strings.ReplaceAll(name, "module.", "")
		dir := mm.FindDir(key)
		if dir == "" {
			fmt.Fprintf(os.Stderr, "No source dir found for %s\n", key)
			continue
		}

		submodules := getRessources(dir, name+".", mm, depth+1, maxDepth)
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
	listOnly := flag.Bool("list", false, "just list the resources, one per line")
	chdir := flag.String("chdir", ".", "lookup resources from this directory")
	maxDepth := flag.Int("depth", 0, "how many levels to decent into submodules")
	flag.Parse()

	if !*listOnly && isatty.IsTerminal(os.Stdout.Fd()) && len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "If used in tty mode, at least one argument must be passed, eg: %s plan\n", os.Args[0])
		os.Exit(1)
	}

	manifest, err := loadManifest(*chdir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		manifest = &ModulesManifest{}
	}
	resources := getRessources(*chdir, "", manifest, 0, *maxDepth)
	if len(resources) == 0 {
		fmt.Fprintf(os.Stderr, "No modules or resources found in %s\n", *chdir)
		os.Exit(1)
	}

	if *listOnly {
		for _, r := range resources {
			fmt.Println(r)
		}
		os.Exit(0)
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
