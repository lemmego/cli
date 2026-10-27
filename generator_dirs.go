package cli

import (
	"fmt"

	"github.com/lemmego/fsys"
)

// ensurePackageDir creates the directory a generated file goes into.
//
// A project says where its code lives, so the target can be somewhere that
// does not exist yet: somebody moving their handlers to ./app/http/handlers
// should not have to create the directory by hand, and without this the write
// fails with a bare "no such file or directory".
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
