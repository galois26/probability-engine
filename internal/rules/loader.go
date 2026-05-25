package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/galois/probability-engine/internal/domain"
)

type Loader struct {
	dir string
}

func NewLoader(dir string) *Loader {
	return &Loader{dir: dir}
}

func (l *Loader) LoadSignalRules(ctx context.Context) ([]domain.SignalRule, error) {
	_ = ctx

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, fmt.Errorf("read rules dir: %w", err)
	}

	out := make([]domain.SignalRule, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".markdown") {
			continue
		}

		path := filepath.Join(l.dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read rule file %s: %w", path, err)
		}

		rule, err := parseRuleMarkdown(string(b))
		if err != nil {
			return nil, fmt.Errorf("parse rule file %s: %w", path, err)
		}
		out = append(out, rule)
	}
	return out, nil
}

func parseRuleMarkdown(s string) (domain.SignalRule, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "---") {
		return domain.SignalRule{}, fmt.Errorf("missing YAML front matter")
	}

	parts := strings.SplitN(s, "---", 3)
	if len(parts) < 3 {
		return domain.SignalRule{}, fmt.Errorf("invalid front matter format")
	}

	var rule domain.SignalRule
	if err := yaml.Unmarshal([]byte(parts[1]), &rule); err != nil {
		return domain.SignalRule{}, fmt.Errorf("unmarshal front matter: %w", err)
	}

	if strings.TrimSpace(rule.Name) == "" {
		return domain.SignalRule{}, fmt.Errorf("rule name is required")
	}
	if rule.Threshold <= 0 {
		rule.Threshold = 0.5
	}
	if rule.DirectionDefault == "" {
		rule.DirectionDefault = domain.DirectionUnknown
	}
	if rule.Labels == nil {
		rule.Labels = map[string]string{}
	}

	return rule, nil
}
