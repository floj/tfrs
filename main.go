package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
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
		return nil, fmt.Errorf("could not load manifest file '%s': %w", path, err)
	}
	defer f.Close()
	var m ModulesManifest
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, fmt.Errorf("could not decode manifest file '%s': %w", path, err)
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

func hasFZF() bool {
	path, err := exec.LookPath("fzf")
	if err != nil {
		return false
	}
	return path != ""
}

func normalize(a []string) []string {
	n := make([]string, 0, len(a))
	for _, e := range a {
		s := strings.TrimSpace(e)
		if s == "" {
			continue
		}
		n = append(n, s)
	}
	return n
}

func pickRessources(names []string) ([]string, error) {
	var buf bytes.Buffer
	cmd := exec.Command("fzf", "-m", `--preview=tr ' ' '\n' <<< '{+}'|sort`)
	cmd.Stdin = strings.NewReader(strings.Join(names, "\n"))
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	return strings.Split(buf.String(), "\n"), nil
}

func main() {
	listOnly := flag.Bool("list", false, "just list the resources, one per line")
	chdir := flag.String("chdir", ".", "lookup resources from this directory")
	maxDepth := flag.Int("depth", 0, "how many levels to decent into submodules")
	prefix := flag.String("prefix", "", "add as a prefix before each selected entry")
	execCmd := flag.String("exec", "", "if set, executes the command using all args and passes the selected, prefixed resources")
	flag.Parse()

	if !hasFZF() {
		fmt.Println("fzf not found in path, please install it. See https://github.com/junegunn/fzf")
		os.Exit(1)
	}

	var err error
	bin := ""

	if *execCmd != "" {
		bin, err = exec.LookPath(*execCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not find %s in PATH: %v", *execCmd, err)
			os.Exit(1)
		}
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
			fmt.Printf("%s%s\n", *prefix, r)
		}
		os.Exit(0)
	}

	names := []string{allMarker}
	names = append(names, resources...)

	selected, err := pickRessources(names)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	selected = normalize(selected)

	for i := 0; i < len(selected); i++ {
		if selected[i] == allMarker {
			selected = []string{}
			break
		}
		selected[i] = fmt.Sprintf("%s%s", *prefix, selected[i])
	}

	if *execCmd == "" {
		fmt.Println(strings.Join(selected, " "))
		return
	}

	var args []string
	args = append(args, *execCmd)
	args = append(args, flag.Args()...)
	args = append(args, selected...)

	fmt.Printf("> %s\n", strings.Join(args, " "))
	err = syscall.Exec(bin, args, os.Environ())
	fmt.Fprintf(os.Stderr, "%v", err)
	os.Exit(1)
}
