package testutil

import "testing"

func SameStrings(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("length mismatch: got=%v want=%v", got, want)
	}

	m := make(map[string]int, len(got))
	for _, v := range got {
		m[v]++
	}
	for _, v := range want {
		m[v]--
	}

	for k, n := range m {
		if n != 0 {
			t.Fatalf("mismatch on %q: got=%v want=%v", k, got, want)
		}
	}
}
