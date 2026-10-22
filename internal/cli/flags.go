package cli

import (
	"flag"
	"strings"
)

func projectFlag(fs *flag.FlagSet) *string {
	project := fs.String("p", "", "project name or ID (default: active project)")
	fs.StringVar(project, "project", "", "project name or ID (default: active project)")
	return project
}

func descriptionFlag(fs *flag.FlagSet) *string {
	description := fs.String("d", "", "description")
	fs.StringVar(description, "description", "", "description")
	return description
}

// setFlags reports which flags were given explicitly, so an empty value can
// still mean "clear it".
func setFlags(fs *flag.FlagSet) map[string]bool {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

// parseOptional parses a command that takes zero or one positional argument.
func parseOptional(fs *flag.FlagSet, args []string, name string) ([]string, error) {
	positional, err := parse(fs, args, 0)
	if err == nil {
		return positional, nil
	}
	return parse(fs, args, 1, name)
}

func containsFold(text, search string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(search))
}
