package version

import "testing"

func TestStringWithoutCommitIsJustTheNumber(t *testing.T) {
	savedVersion, savedCommit := Version, Commit
	defer func() { Version, Commit = savedVersion, savedCommit }()

	Version, Commit = "0.1.0", ""
	if got := String(); got != "0.1.0" {
		t.Fatalf("got %q, want the bare version", got)
	}

	Version, Commit = "0.1.0", "abc1234"
	if got := String(); got != "0.1.0 (abc1234)" {
		t.Fatalf("got %q, want version and commit", got)
	}
}

func TestDevelopmentBuildIsNeverMistakenForARelease(t *testing.T) {
	savedVersion := Version
	defer func() { Version = savedVersion }()
	Version = "0.0.0-dev"
	if String() == "0.1.0" {
		t.Fatal("the development default must not look like a release")
	}
}
