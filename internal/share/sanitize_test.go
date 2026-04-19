package share

import "testing"

func TestSanitizeDisplayName(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"clip.mp4", "clip.mp4", false},
		{"My Video (1).mp4", "My Video (1).mp4", false},
		{"", "", true},
		{"..", "", true},
		{"../clip.mp4", "", true},
		{`a\b.mp4`, "", true},
		{"a/b.mp4", "", true},
		{"clip\nmp4", "", true},
		{"clip\r.mp4", "", true},
		{"clip\x00.mp4", "", true},
		{"CON", "", true},
		{"con.txt", "", true},
		{"COM1.mp4", "", true},
		{"LPT9.dat", "", true},
		{string(make([]byte, 300)), "", true},
	}
	for _, c := range cases {
		got, err := SanitizeDisplayName(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("SanitizeDisplayName(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("SanitizeDisplayName(%q) = %q, %v; want %q, nil", c.in, got, err, c.want)
		}
	}
}
