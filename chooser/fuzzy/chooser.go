package fuzzy

import (
	"github.com/ktr0731/go-fuzzyfinder"
)

type FuzzyChooser struct{}

func NewChooser() *FuzzyChooser {
	return &FuzzyChooser{}
}

func (*FuzzyChooser) Choose(options []string) (bool, []string, error) {
	idx, err := fuzzyfinder.FindMulti(
		options,
		func(i int) string { return options[i] },
	)
	if err != nil {
		return false, nil, err
	}
	var selected []string
	for _, i := range idx {
		selected = append(selected, options[i])
	}
	return true, selected, err
}
