package cli

import (
	"runtime/debug"
)

// version is stamped at build time by the release workflow:
//
//	-ldflags "-X github.com/lemmego/cli.version=v1.2.3"
//
// It is deliberately not a literal with a version in it. One of those sat in
// root.go and went stale at 0.1.59, so every release from v0.1.60 to v0.1.66
// reported a number seven releases old — and nothing failed, because a
// hardcoded string is always valid. The only reliable version is one the
// build supplies.
var version string

// resolveVersion reports the version to print for `lemmego --version`.
//
// Three sources, in descending order of trust:
//
//  1. the ldflags stamp, which is what a released binary carries;
//  2. the module version baked into the binary, which is what `go install
//     github.com/lemmego/cli/cmd@v1.2.3` produces;
//  3. "dev", plus the commit when the toolchain recorded one.
//
// Reporting "dev" for a local build is the point. It is less informative than
// a number and far more useful, because it cannot be wrong.
func resolveVersion() string {
	if version != "" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	// (devel) is what the main module's version reads as for anything but
	// `go install pkg@version` — a local build, or a release binary compiled
	// from a checkout without the stamp.
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}

	var revision, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}
	if revision == "" {
		return "dev"
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if modified == "true" {
		return "dev+" + revision + ".dirty"
	}
	return "dev+" + revision
}
