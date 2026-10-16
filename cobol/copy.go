package cobol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// copybookExtensions are tried in order when looking up a copybook by name.
var copybookExtensions = []string{".cpy", ".CPY", ".cbl"}

// Read lexes a source file and expands its COPY statements with the tokens
// of the copybooks, found by name in the copybook directories. Expanded
// tokens keep the positions they have in their copybook.
func Read(path string, copybookDirs []string) ([]Token, error) {
	return read(path, copybookDirs, nil)
}

func read(path string, copybookDirs []string, including []string) ([]Token, error) {
	if slices.Contains(including, path) {
		return nil, fmt.Errorf("%s: copies itself through %v", path, including)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tokens, err := Lex(path, source)
	if err != nil {
		return nil, err
	}
	var expanded []Token
	for i := 0; i < len(tokens); i++ {
		if !tokens[i].Is("COPY") {
			expanded = append(expanded, tokens[i])
			continue
		}
		name, end, err := parseCopy(tokens, i)
		if err != nil {
			return nil, err
		}
		copybook, err := findCopybook(name, copybookDirs)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", tokens[i].Pos, err)
		}
		copied, err := read(copybook, copybookDirs, append(slices.Clip(including), path))
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, copied...)
		i = end
	}
	return expanded, nil
}

// parseCopy reads COPY name [OF library]. starting at tokens[i] and returns
// the name and the index of the closing period.
func parseCopy(tokens []Token, i int) (string, int, error) {
	pos := tokens[i].Pos
	if i+1 >= len(tokens) || tokens[i+1].Kind != Word {
		return "", 0, fmt.Errorf("%s: COPY without a copybook name", pos)
	}
	name := tokens[i+1].Text
	j := i + 2
	if j+1 < len(tokens) && (tokens[j].Is("OF") || tokens[j].Is("IN")) {
		j += 2
	}
	switch {
	case j < len(tokens) && tokens[j].Is("REPLACING"):
		return "", 0, fmt.Errorf("%s: COPY REPLACING is not supported yet", pos)
	case j >= len(tokens) || tokens[j].Kind != Period:
		return "", 0, fmt.Errorf("%s: COPY %s does not end with a period", pos, name)
	}
	return name, j, nil
}

func findCopybook(name string, dirs []string) (string, error) {
	for _, dir := range dirs {
		for _, ext := range copybookExtensions {
			path := filepath.Join(dir, name+ext)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
		}
	}
	return "", fmt.Errorf("copybook %s not found in %v", name, dirs)
}
