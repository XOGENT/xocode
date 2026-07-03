package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Store reads and writes plan documents under a directory.
type Store struct {
	dir string
}

// NewStore returns a Store rooted at dir (created lazily on Save).
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Dir returns the store's root directory.
func (s *Store) Dir() string { return s.dir }

// Save writes the plan as a Markdown document with a small metadata header and
// fills in p.Slug and p.Path. The filename is <compact-timestamp>-<slug>.md.
func (s *Store) Save(p *Plan) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create plan dir: %w", err)
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	p.Slug = slugify(p.Task)
	name := fmt.Sprintf("%s-%s.md", p.CreatedAt.Format("20060102T150405"), p.Slug)
	p.Path = filepath.Join(s.dir, name)

	doc := renderDoc(p)
	if err := os.WriteFile(p.Path, []byte(doc), 0o644); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	return nil
}

// List returns saved plans newest-first, parsed from the store directory.
// A missing directory is not an error — it just yields no plans.
func (s *Store) List() ([]Plan, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var plans []Plan
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		plans = append(plans, parseDoc(string(b), path))
	}
	// Filenames are timestamp-prefixed, so reverse-lexicographic is newest-first.
	sort.Slice(plans, func(i, j int) bool { return plans[i].Path > plans[j].Path })
	return plans, nil
}

// parseDoc extracts a Plan from a stored document (header + body).
func parseDoc(doc, path string) Plan {
	p := Plan{Path: path, Text: stripHeader(doc)}
	for _, line := range strings.Split(doc, "\n") {
		switch {
		case strings.HasPrefix(line, "- **Task:** "):
			p.Task = strings.TrimSpace(strings.TrimPrefix(line, "- **Task:** "))
		case strings.HasPrefix(line, "- **Model:** "):
			p.Model = strings.TrimSpace(strings.TrimPrefix(line, "- **Model:** "))
		case strings.HasPrefix(line, "- **Created:** "):
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(strings.TrimPrefix(line, "- **Created:** "))); err == nil {
				p.CreatedAt = t
			}
		}
		if strings.HasPrefix(line, "---") {
			break
		}
	}
	return p
}

// Reload re-reads the plan body from disk (used after the user edits it),
// stripping the metadata header so p.Text is just the plan body.
func (s *Store) Reload(p *Plan) error {
	b, err := os.ReadFile(p.Path)
	if err != nil {
		return err
	}
	p.Text = stripHeader(string(b))
	return nil
}

const headerSep = "\n---\n\n"

func renderDoc(p *Plan) string {
	var b strings.Builder
	b.WriteString("# xocode plan\n\n")
	b.WriteString(fmt.Sprintf("- **Task:** %s\n", p.Task))
	if p.Model != "" {
		b.WriteString(fmt.Sprintf("- **Model:** %s\n", p.Model))
	}
	b.WriteString(fmt.Sprintf("- **Created:** %s\n", p.CreatedAt.Format(time.RFC3339)))
	b.WriteString(headerSep)
	b.WriteString(strings.TrimSpace(p.Text))
	b.WriteString("\n")
	return b.String()
}

// stripHeader removes the metadata header (everything up to and including the
// first "---" separator line) so edits to the body round-trip cleanly.
func stripHeader(doc string) string {
	if i := strings.Index(doc, headerSep); i >= 0 {
		return strings.TrimSpace(doc[i+len(headerSep):])
	}
	return strings.TrimSpace(doc)
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slugify produces a filesystem-friendly slug from the task text.
func slugify(task string) string {
	s := strings.ToLower(strings.TrimSpace(task))
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 50 {
		s = strings.Trim(s[:50], "-")
	}
	if s == "" {
		s = "plan"
	}
	return s
}
