package probability

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"
)

func EventFingerprint(e Event) string {
	parts := []string{
		normalize(e.Source),
		normalize(e.Title),
		normalize(e.Summary),
		normalizeURL(e.URL),
		publishedDay(e.PublishedAt),
	}

	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func AssessmentID(eventFingerprint, engineVersion, ruleVersion string) string {
	parts := []string{
		eventFingerprint,
		engineVersion,
		ruleVersion,
	}

	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return normalize(raw)
	}

	u.Fragment = ""
	q := u.Query()
	for _, k := range []string{
		"utm_source",
		"utm_medium",
		"utm_campaign",
		"utm_term",
		"utm_content",
		"fbclid",
		"gclid",
	} {
		q.Del(k)
	}
	u.RawQuery = q.Encode()

	return strings.ToLower(u.String())
}

func publishedDay(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}
