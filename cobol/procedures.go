package cobol

// Span is the paragraphs a procedure reference names, as indexes into the
// program's paragraphs: a paragraph, the paragraphs of a section, or those
// from one through another, through the last of the other's paragraphs if
// it is a section. It is false if a name is neither, or the range runs
// backward.
func (p *Program) Span(ref ProcedureRef) (first, last int, ok bool) {
	first, last, ok = p.procedure(ref.Name)
	if ok && ref.Thru != "" {
		_, last, ok = p.procedure(ref.Thru)
	}
	return first, last, ok && first <= last
}

// procedure is the paragraphs a section or paragraph name names.
func (p *Program) procedure(name string) (first, last int, ok bool) {
	for i, para := range p.Paragraphs {
		if para.Section != name {
			continue
		}
		if !ok {
			first, ok = i, true
		}
		last = i
	}
	if ok {
		return first, last, true
	}
	for i, para := range p.Paragraphs {
		if para.Name == name {
			return i, i, true
		}
	}
	return 0, 0, false
}
