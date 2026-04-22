package admin

import "testing"

func TestParseInfoQuery(t *testing.T) {
	const (
		tok = "aaaabbbbccccddddeeeeffffgg" // 26 chars, lowercase base32
		id  = "abcdefgh"                    // 8 chars, lowercase base32
	)
	cases := []struct {
		name        string
		in          string
		wantToken   string
		wantID      string
		wantErr     bool
	}{
		{"bare token", tok, tok, "", false},
		{"bare id", id, "", id, false},
		{"path-only with name", "/t/" + tok + "/foo.png", tok, "", false},
		{"path-only no name", "/t/" + tok, tok, "", false},
		{"path-only no leading slash", "t/" + tok + "/foo.png", tok, "", false},
		{"full URL", "http://host:8750/t/" + tok + "/foo.png", tok, "", false},
		{"https URL", "https://host/t/" + tok, tok, "", false},
		{"empty", "", "", "", true},
		{"junk", "not a real thing", "", "", true},
		{"URL without /t/", "http://host/other/thing", "", "", true},
		{"token wrong length", "abc", "", "", true},
		{"token wrong chars", "1!!!aaaabbbbccccddddeeefff", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tok, id, err := ParseInfoQuery(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if tok != c.wantToken || id != c.wantID {
				t.Errorf("got token=%q id=%q, want token=%q id=%q",
					tok, id, c.wantToken, c.wantID)
			}
		})
	}
}
