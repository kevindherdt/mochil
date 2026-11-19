// Package version holds the number people see when they install Mochil.
//
// The product has one version, separate from the per-component dates in the changelog: a user
// pinning an image needs "0.1.4" to mean something stable, not a build stamp. The value below is a
// development default; released binaries and images get the real number from the VERSION file at
// build time (-ldflags "-X mochil/internal/version.Version=$(cat VERSION)").
package version

import "fmt"

// Version is the released product version, such as 0.1.0. Left as dev for local builds so a binary
// built from a working tree is never mistaken for a release.
var Version = "0.0.0-dev"

// Commit is the short git commit the binary was built from, when the build supplies it.
var Commit = ""

// String is what -version prints and what the server reports.
func String() string {
	if Commit == "" {
		return Version
	}
	return fmt.Sprintf("%s (%s)", Version, Commit)
}
