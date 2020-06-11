package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
)

func getRessources(dir string) []string {
	module, diags := tfconfig.LoadModule(dir)
	if diags.HasErrors() {
		fmt.Fprintln(os.Stderr, diags)
	}

	modules := make([]string, 0, len(module.ModuleCalls))
	for _, v := range module.ModuleCalls {
		modules = append(modules, "module."+v.Name)
	}

	ressources := make([]string, 0, len(module.ManagedResources))
	for _, v := range module.ManagedResources {
		ressources = append(ressources, v.Type+"."+v.Name)
	}

	sort.Strings(modules)
	sort.Strings(ressources)

	names := make([]string, 0, len(modules)+len(ressources))
	names = append(names, ressources...)
	names = append(names, modules...)
	return names
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <tf command> <opts>\n\n", os.Args[0])
		os.Exit(1)
	}

	names := getRessources(".")
	var question = []*survey.Question{{Prompt: &survey.MultiSelect{
		Message:  "Choose ressources:",
		Options:  names,
		PageSize: len(names),
	}}}

	answers := []string{}

	if err := survey.Ask(question, &answers); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	for i := range answers {
		answers[i] = "-target=" + answers[i]
	}
	fmt.Println(strings.Join(answers, " "))
	var args []string
	args = append(os.Args[1:])
	args = append(args, answers...)

	cmd := exec.Command("terraform", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("> %s\n", cmd)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}
