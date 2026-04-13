package features

import (
	"regexp"
	"sort"
	"strings"

	"probability-engine/internal/domain"
)

type Extractor interface {
	Extract(ev domain.Event) []string
}

type DefaultExtractor struct{}

func NewDefaultExtractor() DefaultExtractor {
	return DefaultExtractor{}
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9:+=_-]+`)

var stopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {},
	"from": {}, "into": {}, "over": {}, "under": {}, "after": {}, "before": {},
	"about": {}, "across": {}, "against": {}, "among": {}, "around": {},
	"because": {}, "between": {}, "during": {}, "through": {}, "while": {},
	"market": {}, "markets": {}, "business": {}, "report": {}, "reports": {},
	"news": {}, "article": {}, "updated": {}, "published": {}, "says": {},
	"said": {}, "according": {}, "today": {}, "worth": {}, "watching": {},
	"new": {}, "latest": {}, "post": {}, "posts": {}, "story": {}, "stories": {},
}

var lowSignalTokens = map[string]struct{}{
	// generic finance/news noise
	"price": {}, "prices": {}, "stock": {}, "stocks": {}, "shares": {}, "trading": {},
	"investors": {}, "investor": {}, "analyst": {}, "analysts": {}, "company": {},
	"companies": {}, "asset": {}, "assets": {},

	// crypto/promotional noise that was contaminating supply_chain
	"bitcoin": {}, "crypto": {}, "ethereum": {}, "binance": {}, "token": {}, "tokens": {},
	"etf": {}, "etfs": {}, "blockchain": {}, "meme": {},

	// generic web/source boilerplate
	"source": {}, "image": {}, "featured": {}, "unsplash": {},
}

var allowedLabelTokens = map[string]struct{}{
	"label:category=geopolitics": {},
	"label:category=energy":      {},
	"label:category=trade":       {},
	//"label:category=crypto":      {},
}

func (DefaultExtractor) Extract(ev domain.Event) []string {
	raw := strings.Join([]string{
		ev.Title,
		ev.Summary,
	}, " ")

	tokens := tokenize(raw)

	out := make([]string, 0, len(tokens)+len(ev.Labels))
	seen := make(map[string]struct{})

	for _, tok := range tokens {
		tok = normalizeToken(tok)
		if !keepToken(tok) {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}

	for k, v := range ev.Labels {
		tok := "label:" + strings.ToLower(strings.TrimSpace(k)) + "=" + strings.ToLower(strings.TrimSpace(v))
		if _, ok := allowedLabelTokens[tok]; !ok {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}

	sort.Strings(out)
	return out
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, " ")
	return strings.Fields(s)
}

func normalizeToken(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func keepToken(s string) bool {
	if len(s) < 3 {
		return false
	}
	if isNumericOnly(s) {
		return false
	}
	if containsDigit(s) {
		return false
	}
	if _, ok := stopwords[s]; ok {
		return false
	}
	if _, ok := lowSignalTokens[s]; ok {
		return false
	}
	if strings.HasPrefix(s, "source:") {
		return false
	}
	if strings.HasPrefix(s, "label:") {
		_, ok := allowedLabelTokens[s]
		return ok
	}
	return true
}

func isNumericOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
