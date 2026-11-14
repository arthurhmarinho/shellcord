package event

import "testing"

func TestProgName(t *testing.T) {
	cases := map[string]string{
		"nvim main.go":                  "nvim",
		"/usr/bin/git push origin main": "git",
		"  ./build.sh --release ":       "build.sh",
		"sudo nvim /etc/fstab":          "nvim",
		"sudo -u postgres psql":         "psql",
		"sudo -E -- make install":       "make",
		"doas -u root vi x":             "vi",
		"env FOO=1 BAR=2 cargo build":   "cargo",
		"env -u HOME go test":           "go",
		"FOO=1 cargo run":               "cargo",
		"time nice -n 10 make -j8":      "make",
		"nohup command ./server &":      "server",
		"exec nvim":                     "nvim",
		"ls; echo x":                    "ls",
		"cat a | grep b":                "cat",
		"sudo":                          "sudo",
		"sudo -i":                       "sudo",
		"FOO=1":                         "",
		"":                              "",
		"   ":                           "",
	}
	for in, want := range cases {
		if got := ProgName(in); got != want {
			t.Errorf("ProgName(%q) = %q, quero %q", in, got, want)
		}
	}
}

func TestProgNameIdempotent(t *testing.T) {
	for _, in := range []string{"sudo nvim", "env", "FOO=1 x", "a=b", "-x", "time"} {
		p := ProgName(in)
		if ProgName(p) != p {
			t.Errorf("ProgName(%q) = %q não é estável: ProgName(%q) = %q", in, p, p, ProgName(p))
		}
	}
}

func TestRoundTrip(t *testing.T) {
	e := Event{V: Version, Src: SourceFish, PID: 42, Start: 7, Kind: KindExec, Prog: "nvim", Cwd: "/home/x/proj"}
	b, err := Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if got != e {
		t.Fatalf("got %+v, quero %+v", got, e)
	}
}

func TestValidateRejects(t *testing.T) {
	bad := []Event{
		{V: 2, Src: SourceFish, PID: 1, Kind: KindIdle},
		{V: 1, Src: "vim", PID: 1, Kind: KindIdle},
		{V: 1, Src: SourceFish, Kind: KindIdle},                           // sem pid
		{V: 1, Src: SourceFish, PID: 1, Kind: KindBusy},                   // tipo do claude
		{V: 1, Src: SourceFish, PID: 1, Kind: KindExec},                   // exec sem prog
		{V: 1, Src: SourceFish, PID: 1, Kind: KindExec, Prog: "git push"}, // argumentos
		{V: 1, Src: SourceClaude, Kind: KindBusy},                         // sem session_id
		{V: 1, Src: SourceClaude, ID: "s", Kind: KindExec},                // tipo do fish
		{V: 1, Src: SourceClaude, ID: "s", Kind: KindIdle, Tool: "Bash"},  // tool sem busy
		{V: 1, Src: SourceFish, PID: 1, Kind: KindExec, Prog: "x", Tool: "Bash"},
		{V: 1, Src: SourceSystem, Kind: KindIdle},
		{V: 1, Src: SourceSystem, Kind: KindAway, Cwd: "/x"},
		{V: 1, Src: SourceFish, PID: 1, Kind: KindAway},
	}
	for _, e := range bad {
		if e.Validate() == nil {
			t.Errorf("Validate(%+v) aceitou um evento inválido", e)
		}
	}
}

func TestDecodeIgnoresUnknownFields(t *testing.T) {
	_, err := Decode([]byte(`{"v":1,"src":"claude","id":"s","kind":"busy","futuro":true}`))
	if err != nil {
		t.Fatal(err)
	}
}
