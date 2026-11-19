package narration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The worker's text cleanup decides what the voice is given, so it is checked
// with the real function. It needs only Python's standard library.
func TestWorkerCleansTextForTheVoice(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}
	script := filepath.Join(t.TempDir(), "worker.py")
	if err := os.WriteFile(script, []byte(workerScript), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"THE SUBTLE KNIFE":                       "The subtle knife.",
		"NASA":                                   "NASA.",
		"nobody -- someone's bastard":            "nobody, someone's bastard.",
		"She read _Madam, Will You Talk?_ twice": "She read Madam, Will You Talk? twice.",
		"[Tips] Mensch are a race":               "Tips Mensch are a race.",
		"“Do not,” she said, “look back.”":       "“Do not,” she said, “look back.”",
		"He waited:":                             "He waited.",
		"* * *":                                  "",
		"   ":                                    "",
		"A  line\twith   odd spacing":            "A line with odd spacing.",
	}
	input, _ := json.Marshal(cases)
	program := `
import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("worker", sys.argv[1])
worker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(worker)
cases = json.loads(sys.stdin.read())
sys.__stdout__.write(json.dumps({text: worker.speakable(text) for text in cases}))
`
	cmd := exec.Command(python, "-c", program, script)
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running the worker's cleanup: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	for text, want := range cases {
		if got[text] != want {
			t.Errorf("speakable(%q) = %q, want %q", text, got[text], want)
		}
	}
}
