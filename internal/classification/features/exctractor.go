package features

import (
	"regexp"
	"strings"

	"probability-engine/internal/domain"
)

type Extractor interface {
	Extract(ev domain.Event) []string
}

type DefaultExtractor struct {
	nonWord *regexp.Regexp
}

func NewDefaultExtractor() *DefaultExtractor {
	return &DefaultExtractor{
		nonWord: regexp.MustCompile(`[^a-zA-Z0-9]+`),
	}
}

func (e *DefaultExtractor) Extract(ev domain.Event) []string {
	text := strings.ToLower(strings.TrimSpace(ev.Title + " " + ev.Summary))
	text = e.nonWord.ReplaceAllString(text, " ")
	parts := strings.Fields(text)

	out := make([]string, 0, len(parts)+len(ev.Labels)+4)
	seen := make(map[string]struct{}, len(parts)+len(ev.Labels))

	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	for _, p := range parts {
		add(p)
	}
	if ev.Source != "" {
		add("source:" + strings.ToLower(ev.Source))
	}
	if ev.Country != "" {
		add("country:" + strings.ToLower(ev.Country))
	}
	for k, v := range ev.Labels {
		add("label:" + strings.ToLower(k) + "=" + strings.ToLower(v))
	}

	return out
}
