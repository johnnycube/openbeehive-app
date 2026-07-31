package sqlstore

import (
	"sort"
	"testing"
)

func TestPhotoKeyElements(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"orset with live and removed elements",
			`{"k1":{"a":["t1"],"r":[]},"k2":{"a":["t2"],"r":["t2"]}}`,
			[]string{"k1", "k2"}}, // removed elements included: their blobs linger too
		{"legacy plain array", `["k1","k2"]`, []string{"k1", "k2"}},
		{"empty", ``, nil},
		{"null", `null`, nil},
		{"garbage", `not json`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := photoKeyElements(c.raw)
			sort.Strings(got)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("got %v, want %v", got, c.want)
				}
			}
		})
	}
}
