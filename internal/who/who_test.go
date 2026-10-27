package who_test

import (
	"testing"

	"github.com/janearc/puzzlemouth/internal/who"
)

func TestOf(t *testing.T) {
	cases := []struct {
		who, sids, signer, want string
	}{
		{"", "", "", "gort is not signed\n" +
			"gort was built by go rather than by game: " +
			"no builder and no sessions in it"},
		{"sid/1a2b3c4d on box", "sid/1a2b3c4d,sid/5e6f7a8b", "",
			"gort is not signed\n" +
				"gort was built by sid/1a2b3c4d on box\n" +
				"the last sessions to commit: sid/1a2b3c4d, sid/5e6f7a8b"},
		{"no session on box", "", "ada tester",
			"gort was signed by ada tester, as gort itself checked;\n" +
				"for proof, check gort.sig with ssh-keygen -Y verify\n" +
				"gort was built by no session on box\n" +
				"no session signed the commits behind it"},
	}
	for _, example := range cases {
		got := who.Of("gort", example.who, example.sids, example.signer)
		if got != example.want {
			t.Errorf("Of(%q, %q, %q) =\n%s\nwant\n%s", example.who,
				example.sids, example.signer, got, example.want)
		}
	}
}
