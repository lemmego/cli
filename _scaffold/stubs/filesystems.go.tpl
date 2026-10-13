package configs

import (
	"github.com/lemmego/api/config"
)

func init() {
	// Two local disks, split by whether what is on them may be served.
	//
	//	storage/
	//	  app/
	//	    public/      <- served at /storage/ by fs.Provider
	//	    private/     <- the default disk; never served
	//	  logs/
	//	  session/
	//	  database.sqlite
	//
	// The split is the security property. One disk rooted at storage means
	// serving uploads from a directory that also holds the logs — and the
	// log is where mail is written outside production, so it carries live
	// email-confirmation and password-reset links under a fixed filename.
	// With two disks, what is public is where the file is rather than a list
	// somebody has to remember not to widen.
	//
	// STORAGE_PATH should be absolute in a real deployment: a relative path
	// resolves against the working directory, so a single binary started from
	// anywhere but the project root writes uploads somewhere else and serves
	// nothing, with a 404 as the only symptom. fs.Provider logs the root it
	// resolved at boot for exactly that reason.
	root := config.MustEnv("STORAGE_PATH", "./storage")

	config.Set("filesystems", config.M{
		"default": config.MustEnv("FILESYSTEM_DISK", "local"),
		"disks": config.M{
			// Not served. The right default, because a mistake fails closed.
			"local": config.M{
				"driver": "local",
				"root":   root + "/app/private",
				"path":   root + "/app/private",
			},
			// Served in full at /storage/. Write nothing here that should not
			// be readable by anybody with the URL, and reach it with
			// c.UploadOn("public", ...) or c.StoreUploadOn("public", ...).
			"public": config.M{
				"driver": "local",
				"root":   root + "/app/public",
				"path":   root + "/app/public",
			},
{{- if eq .FilesystemDisk "s3"}}
			"s3": config.M{
				"driver":   "s3",
				"key":      config.MustEnv("AWS_ACCESS_KEY_ID", ""),
				"secret":   config.MustEnv("AWS_SECRET_ACCESS_KEY", ""),
				"region":   config.MustEnv("AWS_DEFAULT_REGION", "us-east-1"),
				"bucket":   config.MustEnv("AWS_BUCKET", ""),
				"endpoint": config.MustEnv("AWS_ENDPOINT", ""),
			},
{{- end}}
		},
	})
}
