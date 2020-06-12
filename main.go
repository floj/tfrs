package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"golang.org/x/crypto/ssh/terminal"
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

func loadConfig() (Config, error) {
	c := Config{}
	dir, err := os.UserConfigDir()
	if err != nil {
		return c, fmt.Errorf("Could not detect config dir: %w", err)
	}
	path := filepath.Join(dir, "tfrs", "config.json")
	f, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		return c, nil
	} else if err != nil {
		return c, fmt.Errorf("Could not open config file %s: %w", path, err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&c); err != nil {
		return c, fmt.Errorf("Could not parse config file %s: %w", path, err)
	}

	return c, nil
}

type Config struct {
	Input       survey.IconSet `json:"input"`
	SpaceAdjust string         `json:"space_adjust"`
}

func applyIcon(source survey.Icon, dest *survey.Icon) {
	if source.Text != "" {
		dest.Text = source.Text
	}
	if source.Format != "" {
		dest.Format = source.Format
	}
}

func (c *Config) applyIcons(icons *survey.IconSet) {
	applyIcon(c.Input.Error, &icons.Error)
	applyIcon(c.Input.Help, &icons.Help)
	applyIcon(c.Input.HelpInput, &icons.HelpInput)
	applyIcon(c.Input.MarkedOption, &icons.MarkedOption)
	applyIcon(c.Input.Question, &icons.Question)
	applyIcon(c.Input.SelectFocus, &icons.SelectFocus)
	applyIcon(c.Input.UnmarkedOption, &icons.UnmarkedOption)
}

func main() {

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <tf command> <opts>\n\n", os.Args[0])
		os.Exit(1)
	}

	conf, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	names := getRessources(".")

	pageSize := len(names)
	_, height, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err == nil && height < pageSize {
		pageSize = height - 2
	}

	var question = []*survey.Question{{Prompt: &survey.MultiSelect{
		Message:  "Choose ressources/modules:",
		Options:  names,
		PageSize: pageSize,
	}}}

	answers := []string{}
	spaceAdjust := " "
	if conf.SpaceAdjust != "" {
		spaceAdjust = conf.SpaceAdjust
	}

	survey.MultiSelectQuestionTemplate = `
	{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
	{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
	{{- color "default+hb"}}{{ .Message }}{{ .FilterMessage }}{{color "reset"}}
	{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
	{{- else }}
		{{- "  "}}{{- color "cyan"}}[Use arrows to move, space to select, type to filter{{- if and .Help (not .ShowHelp)}}, {{ .Config.HelpInput }} for more help{{end}}]{{color "reset"}}
	  {{- "\n"}}
	  {{- range $ix, $option := .PageEntries}}
		{{- if eq $ix $.SelectedIndex }}{{color $.Config.Icons.SelectFocus.Format }}{{ $.Config.Icons.SelectFocus.Text }}{{color "reset"}}{{else}}` + spaceAdjust + `{{end}}
		{{- if index $.Checked $option.Index }}{{color $.Config.Icons.MarkedOption.Format }} {{ $.Config.Icons.MarkedOption.Text }} {{else}}{{color $.Config.Icons.UnmarkedOption.Format }} {{ $.Config.Icons.UnmarkedOption.Text }} {{end}}
		{{- color "reset"}}
		{{- " "}}{{$option.Value}}{{"\n"}}
	  {{- end}}
	{{- end}}`

	if err := survey.Ask(question, &answers, survey.WithIcons(conf.applyIcons)); err != nil {
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
	if err = cmd.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
