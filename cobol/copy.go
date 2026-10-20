package cobol

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// CopybookExtensions are tried in order when looking up a copybook by name.
var CopybookExtensions = []string{".cpy", ".CPY", ".cbl"}

// Read lexes a source file and expands its COPY statements with the tokens
// of the copybooks, found by name in the copybook directories: a name
// quoted as a literal is the file's name as written, and a word is tried
// with each of the extensions. Expanded
// tokens keep the positions they have in their copybook.
//
// COPY ... REPLACING replaces text in the copied tokens: pseudo-text
// (==text==), a word or a literal replaces the same tokens wherever they
// occur in sequence. A word tagged with colons, such as ==:P:==, also
// replaces the tag inside words: with REPLACING ==:P:== BY ==WS==, the
// copybook's :P:-COUNT becomes WS-COUNT.
//
// A REPLACE statement in the program replaces text the same way in all
// the source after it, copybooks expanded, until REPLACE OFF or the next
// REPLACE.
func Read(path string, copybookDirs []string) ([]Token, error) {
	return ReadIn(path, copybookDirs, "")
}

// ReadIn reads a source file in an encoding, as LexIn lexes it.
func ReadIn(path string, copybookDirs []string, encoding string) ([]Token, error) {
	return read(path, copybookDirs, encoding, nil)
}

func read(path string, copybookDirs []string, encoding string, including []string) ([]Token, error) {
	if slices.Contains(including, path) {
		return nil, fmt.Errorf("%s: copies itself through %v", path, including)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tokens, err := LexIn(path, source, encoding)
	if err != nil {
		return nil, err
	}
	var expanded []Token
	for i := 0; i < len(tokens); i++ {
		if !tokens[i].Is("COPY") {
			expanded = append(expanded, tokens[i])
			continue
		}
		name, replacements, end, err := parseCopy(tokens, i)
		if err != nil {
			return nil, err
		}
		copybook, err := findCopybook(name, copybookDirs)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", tokens[i].Pos, err)
		}
		copied, err := read(copybook, copybookDirs, encoding, append(slices.Clip(including), path))
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, replace(copied, replacements)...)
		i = end
	}
	if including != nil {
		return expanded, nil
	}
	return applyReplace(expanded)
}

// replacement is one pair of a COPY's REPLACING phrase.
type replacement struct {
	from, to []Token
}

// parseCopy reads COPY name [OF library] [REPLACING from BY to ...].
// starting at tokens[i] and returns the name, the replacements and the
// index of the closing period.
func parseCopy(tokens []Token, i int) (string, []replacement, int, error) {
	pos := tokens[i].Pos
	if i+1 >= len(tokens) || tokens[i+1].Kind != Word && tokens[i+1].Kind != String {
		return "", nil, 0, fmt.Errorf("%s: COPY without a copybook name", pos)
	}
	name := tokens[i+1].Text
	j := i + 2
	if j+1 < len(tokens) && (tokens[j].Is("OF") || tokens[j].Is("IN")) {
		j += 2
	}
	var replacements []replacement
	if j < len(tokens) && tokens[j].Is("REPLACING") {
		var err error
		if replacements, j, err = parseReplacements(tokens, j+1, "COPY "+name+" REPLACING"); err != nil {
			return "", nil, 0, err
		}
	}
	if j >= len(tokens) || tokens[j].Kind != Period {
		return "", nil, 0, fmt.Errorf("%s: COPY %s does not end with a period", pos, name)
	}
	return name, replacements, j, nil
}

// parseReplacements reads the pairs of a REPLACING phrase or a REPLACE
// statement, from BY to ..., starting at tokens[j], up to the period, and
// returns them and the period's index.
func parseReplacements(tokens []Token, j int, what string) ([]replacement, int, error) {
	pos := tokens[j-1].Pos
	var replacements []replacement
	for j < len(tokens) && tokens[j].Kind != Period {
		var r replacement
		var err error
		if r.from, j, err = operand(tokens, j); err != nil {
			return nil, 0, err
		}
		if j >= len(tokens) || !tokens[j].Is("BY") {
			return nil, 0, fmt.Errorf("%s: %s without BY", pos, what)
		}
		if r.to, j, err = operand(tokens, j+1); err != nil {
			return nil, 0, err
		}
		if len(r.from) == 0 {
			return nil, 0, fmt.Errorf("%s: %s empty text", pos, what)
		}
		replacements = append(replacements, r)
	}
	return replacements, j, nil
}

// applyReplace applies each REPLACE statement among the tokens to those
// after it, up to the next: REPLACE ==from== BY ==to== ..., as COPY
// REPLACING replaces, until REPLACE OFF. The statements themselves are
// left out.
func applyReplace(tokens []Token) ([]Token, error) {
	var out []Token
	var active []replacement
	start := 0
	for i := 0; i < len(tokens); i++ {
		if !tokens[i].Is("REPLACE") || i+1 >= len(tokens) || !tokens[i+1].Is("OFF") && !strings.HasPrefix(tokens[i+1].Text, "==") {
			continue
		}
		out = append(out, replace(tokens[start:i], active)...)
		j := i + 1
		if tokens[j].Is("OFF") {
			active, j = nil, j+1
		} else {
			var err error
			if active, j, err = parseReplacements(tokens, j, "REPLACE"); err != nil {
				return nil, err
			}
		}
		if j >= len(tokens) || tokens[j].Kind != Period {
			return nil, fmt.Errorf("%s: REPLACE does not end with a period", tokens[i].Pos)
		}
		start, i = j+1, j
	}
	return append(out, replace(tokens[start:], active)...), nil
}

// operand reads one operand of a REPLACING phrase starting at tokens[j]: a
// pseudo-text between == delimiters, a literal, or a word with the OF or
// IN names that qualify it. It returns the operand's tokens and the index
// after it.
func operand(tokens []Token, j int) ([]Token, int, error) {
	if j >= len(tokens) {
		return nil, j, fmt.Errorf("COPY REPLACING ends early")
	}
	first := tokens[j]
	if first.Kind != Word || !strings.HasPrefix(first.Text, "==") {
		end := j + 1
		for first.Kind == Word && end+1 < len(tokens) && (tokens[end].Is("OF") || tokens[end].Is("IN")) {
			end += 2
		}
		if first.Kind == Word && end < len(tokens) && tokens[end].Kind == LParen {
			depth := 0
			for ; end < len(tokens); end++ {
				switch tokens[end].Kind {
				case LParen:
					depth++
				case RParen:
					depth--
				}
				if depth == 0 {
					end++
					break
				}
			}
		}
		return tokens[j:end], end, nil
	}
	var text []Token
	for k := j; k < len(tokens); k++ {
		t := tokens[k]
		if k == j {
			t.Text = strings.TrimPrefix(t.Text, "==")
		}
		closes := t.Kind == Word && strings.HasSuffix(t.Text, "==") && (k > j || len(first.Text) >= 4)
		if closes {
			t.Text = strings.TrimSuffix(t.Text, "==")
		}
		if t.Text != "" {
			text = append(text, t)
		}
		if closes {
			return text, k + 1, nil
		}
	}
	return nil, j, fmt.Errorf("%s: pseudo-text without its closing ==", first.Pos)
}

// replace applies the replacements to copied tokens, each token replaced at
// most once: whole sequences of tokens first, then colon tags inside words.
// Replacing tokens take the position of the text they replace, and its
// place in area A, so that a paragraph name replaced is still a header.
func replace(tokens []Token, replacements []replacement) []Token {
	if len(replacements) == 0 {
		return tokens
	}
	var out []Token
	for i := 0; i < len(tokens); {
		if r, ok := matching(tokens[i:], replacements); ok {
			for _, t := range r.to {
				t.Pos, t.Comment, t.AreaA = tokens[i].Pos, tokens[i].Comment, tokens[i].AreaA
				out = append(out, t)
			}
			i += len(r.from)
			continue
		}
		out = append(out, retag(tokens[i], replacements))
		i++
	}
	return out
}

// matching finds the replacement whose text the tokens start with.
func matching(tokens []Token, replacements []replacement) (replacement, bool) {
	for _, r := range replacements {
		if len(r.from) <= len(tokens) && slices.EqualFunc(r.from, tokens[:len(r.from)], sameText) {
			return r, true
		}
	}
	return replacement{}, false
}

func sameText(a, b Token) bool { return a.Kind == b.Kind && a.Text == b.Text }

// retag replaces colon tags, :TAG:, inside a word.
func retag(t Token, replacements []replacement) Token {
	for _, r := range replacements {
		if t.Kind != Word || len(r.from) != 1 || len(r.to) > 1 || !isTag(r.from[0].Text) {
			continue
		}
		to := ""
		if len(r.to) == 1 {
			to = r.to[0].Text
		}
		t.Text = strings.ReplaceAll(t.Text, r.from[0].Text, to)
	}
	return t
}

func isTag(word string) bool {
	return len(word) > 2 && strings.HasPrefix(word, ":") && strings.HasSuffix(word, ":")
}

func findCopybook(name string, dirs []string) (string, error) {
	for _, dir := range dirs {
		for _, ext := range append([]string{""}, CopybookExtensions...) {
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
