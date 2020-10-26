package chooser

type Chooser interface {
	Choose(options []string) (bool, []string, error)
}
