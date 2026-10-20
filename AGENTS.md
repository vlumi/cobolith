# Agent instructions

Cobolith is a Go library for reading COBOL: `cobol` lexes and parses source and copybooks, `layout` lays out records in a dialect, `record` reads and writes their fields, and `jcl` reads the jobs that run them. `conformance` is a command that reads NIST's COBOL-85 test suite with it and writes a report. README.md says what it reads and what it does not yet.

## Rules

- Never mention where this library came from, any company or product it might be compared to as its origin, or any hiring process, in code, documentation, commit messages or anywhere else in the repository. Technical references to well-known tools and systems (GnuCOBOL, IBM and Micro Focus COBOL, NIST's suite) are fine where they explain a choice.
- What it reads, it reads exactly: a record's layout is the one the compiler makes in that dialect, or an error. Something it does not read is an error that says what, with the file and line, never a guess.
- Its tests read only their own `testdata` and fixtures written for them, never a real system's code or data. NIST's suite is fetched by `make conformance`, never committed; its terms ask for an explicit acknowledgment of NIST's work, which README.md and the report give.
- A change that changes what the suite's programs read as updates `conformance/README.md` in the same commit: run `make conformance`.
- All text is English with en-US spelling: code, comments, documentation and commit messages. Comments are minimal.
- Markdown paragraphs are one line each; the editor wraps. `.markdownlint.jsonc` encodes this.
- Routine operations are make targets with a `##` description, so plain `make` lists them; scripts do the work, the Makefile names it.

## Commits and pull requests

- The history reads as what actually happened. Commits are granular: each one an atomic change, or a few closely related ones, that builds and passes on its own. Something found later gets its own commit saying what it fixes, rather than being folded back into history.
- Commit messages describe the change, not the process.
- Every chunk of work is a branch and a pull request against `main`, started from current `main`. Pushed commits are not rewritten: no amending, squashing or force-pushing. `main` is never rewritten and never committed to directly.
- Run `make ci` before opening or updating a pull request and only push when it passes; a change to Markdown alone needs none.
- Merge commits only, so the by-concern commits stay visible on `main`; branches are deleted on merge: `gh pr merge --merge --delete-branch` once the hosted check has passed.
- Releases are tags, `vMAJOR.MINOR.PATCH`, below 1.0 while the API moves.

## Code

- The code should read well to a stranger: idiomatic Go, small functions with names that say what they do, no dead code, no speculative abstractions. The exported API is what its users need, documented, each package with a runnable example.
- Every construct read is covered by tests: the tables of statement forms and of data description entries say what each reads as, in each dialect it differs in. Linters and formatters run in `make ci` and stay clean.
