// Command assetbuild bundles the Mochil web shell for production.
//
// It turns the ~60 small ES modules and 13 stylesheets that index.html loads
// into one content-hashed script (plus lazily loaded chunks) and one content-
// hashed stylesheet under internal/web/static/dist. The server swaps them into
// index.html when dist/manifest.json exists; without it the unbundled sources
// are served as before, so development needs no build step.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

type manifest struct {
	Script  string   `json:"script"`
	Entry   string   `json:"entry"`
	Style   string   `json:"style"`
	Styles  []string `json:"styles"`
	Preload []string `json:"preload"`
}

var (
	stylesheetLink = regexp.MustCompile(`<link rel="stylesheet" href="/assets/([^"?]+)(?:\?[^"]*)?"\s*/?>`)
	entryScript    = regexp.MustCompile(`<script type="module" src="/assets/([^"?]+)(?:\?[^"]*)?"></script>`)
)

func main() {
	staticDir := flag.String("static", "internal/web/static", "static asset directory")
	flag.Parse()
	if err := run(*staticDir); err != nil {
		fmt.Fprintln(os.Stderr, "assetbuild:", err)
		os.Exit(1)
	}
}

func run(staticDir string) error {
	staticDir, err := filepath.Abs(staticDir)
	if err != nil {
		return err
	}
	index, err := os.ReadFile(filepath.Join(staticDir, "index.html"))
	if err != nil {
		return err
	}
	script := entryScript.FindSubmatch(index)
	if script == nil {
		return fmt.Errorf("index.html has no module entry script")
	}
	var styles []string
	for _, match := range stylesheetLink.FindAllSubmatch(index, -1) {
		styles = append(styles, string(match[1]))
	}
	dist := filepath.Join(staticDir, "dist")
	if err := os.RemoveAll(dist); err != nil {
		return err
	}
	if err := os.MkdirAll(dist, 0o755); err != nil {
		return err
	}

	built := manifest{Styles: styles, Entry: string(script[1])}
	if err := buildScripts(staticDir, dist, &built); err != nil {
		return err
	}
	if err := buildStyles(staticDir, dist, styles, &built); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(built, "", "  ")
	return os.WriteFile(filepath.Join(dist, "manifest.json"), data, 0o644)
}

// resolver maps cache-busting "?v=N" imports and absolute /assets/ imports to
// real files so each module is bundled exactly once.
func resolver(staticDir string) api.Plugin {
	return api.Plugin{
		Name: "bobobooks-paths",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^\.{1,2}/.*\?`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				clean := args.Path[:strings.Index(args.Path, "?")]
				return api.OnResolveResult{Path: filepath.Join(args.ResolveDir, clean)}, nil
			})
			build.OnResolve(api.OnResolveOptions{Filter: `^/assets/`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				clean := strings.TrimPrefix(args.Path, "/assets/")
				if i := strings.Index(clean, "?"); i >= 0 {
					clean = clean[:i]
				}
				if strings.HasSuffix(clean, ".css") || !strings.HasSuffix(clean, ".js") {
					return api.OnResolveResult{Path: args.Path, External: true}, nil
				}
				return api.OnResolveResult{Path: filepath.Join(staticDir, clean)}, nil
			})
		},
	}
}

func buildScripts(staticDir, dist string, out *manifest) error {
	result := api.Build(api.BuildOptions{
		EntryPointsAdvanced: []api.EntryPoint{{OutputPath: "app", InputPath: filepath.Join(staticDir, out.Entry)}},
		Bundle:              true,
		Splitting:           true,
		Format:              api.FormatESModule,
		Target:              api.ES2022,
		Outdir:              dist,
		Outbase:             staticDir,
		EntryNames:          "[name]-[hash]",
		ChunkNames:          "chunks/[name]-[hash]",
		MinifyWhitespace:    true,
		MinifySyntax:        true,
		MinifyIdentifiers:   true,
		Metafile:            true,
		Write:               true,
		LogLevel:            api.LogLevelWarning,
		Plugins:             []api.Plugin{resolver(staticDir)},
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("script bundle failed: %s", result.Errors[0].Text)
	}
	var meta struct {
		Outputs map[string]struct {
			EntryPoint string `json:"entryPoint"`
			Imports    []struct {
				Path string `json:"path"`
				Kind string `json:"kind"`
			} `json:"imports"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(result.Metafile), &meta); err != nil {
		return err
	}
	rel := func(path string) string {
		abs, _ := filepath.Abs(path)
		value, _ := filepath.Rel(staticDir, abs)
		return filepath.ToSlash(value)
	}
	for path, output := range meta.Outputs {
		if output.EntryPoint == "" || filepath.Base(output.EntryPoint) != filepath.Base(out.Entry) {
			continue
		}
		out.Script = rel(path)
		seen := map[string]bool{}
		for _, imported := range output.Imports {
			if name := rel(imported.Path); imported.Kind == "import-statement" && !seen[name] {
				seen[name] = true
				out.Preload = append(out.Preload, name)
			}
		}
	}
	sort.Strings(out.Preload)
	if out.Script == "" {
		return fmt.Errorf("bundle produced no entry script")
	}
	return nil
}

func buildStyles(staticDir, dist string, styles []string, out *manifest) error {
	var imports strings.Builder
	for _, name := range styles {
		fmt.Fprintf(&imports, "@import %q;\n", filepath.Join(staticDir, name))
	}
	result := api.Build(api.BuildOptions{
		Stdin:            &api.StdinOptions{Contents: imports.String(), ResolveDir: staticDir, Loader: api.LoaderCSS, Sourcefile: "core.css"},
		Bundle:           true,
		Outfile:          filepath.Join(dist, "core.css"),
		MinifyWhitespace: true,
		MinifySyntax:     true,
		Write:            true,
		LogLevel:         api.LogLevelWarning,
		Plugins:          []api.Plugin{resolver(staticDir)},
		External:         []string{"/assets/*"},
		Metafile:         false,
		AllowOverwrite:   true,
		LegalComments:    api.LegalCommentsNone,
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("style bundle failed: %s", result.Errors[0].Text)
	}
	data, err := os.ReadFile(filepath.Join(dist, "core.css"))
	if err != nil {
		return err
	}
	name := fmt.Sprintf("core-%x.css", hashBytes(data))
	if err := os.Rename(filepath.Join(dist, "core.css"), filepath.Join(dist, name)); err != nil {
		return err
	}
	out.Style = "dist/" + name
	return nil
}

func hashBytes(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:6]
}
