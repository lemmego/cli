package cli

import (
	"fmt"

	"github.com/lemmego/fsys"
)

// ensurePackageDir creates the directory a generated file goes into.
//
// It did not used to matter: every generator wrote into a directory the
// scaffold had already created. Now that a project says where its code lives,
// the answer can be somewhere that does not exist yet — a project moving its
// handlers to ./app/http/handlers should not have to create the directory by
// hand first, and the alternative is a write failing with a bare
// "no such file or directory".
func ensurePackageDir(fs fsys.FS, path string) error {
	if path == "" {
		return fmt.Errorf("no output directory was resolved")
	}
	if exists, _ := fs.Exists(path); exists {
		return nil
	}
	if err := fs.CreateDirectory(path); err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	return nil
}
