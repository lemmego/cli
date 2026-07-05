package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ScanStr scans the given input
func ScanStr(input *string, label string) {
	fmt.Print(fmt.Sprintf("Enter the %s name: ", label))
	_, err := fmt.Scanln(input)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error reading %s name:", label), err)
	}
}

// CopyFile is a helper function to copy a file
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}
	//err = os.Chmod(dst, 0644) // Set the file permissions
	//if err != nil {
	//	return err
	//}
	//
	// Copy file info (like modification times)
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
}

// CopyDir copies the src dir to dst
func CopyDir(src string, dst string) error {
	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(sourcePath, destPath)
			if err != nil {
				return err
			}
		} else {
			// Check if the destination file exists
			if _, err := os.Stat(destPath); err == nil {
				// If the file exists, remove it before copying
				if err := os.Remove(destPath); err != nil {
					return fmt.Errorf("failed to remove existing destination file: %v", err)
				}
			}
			err = CopyFile(sourcePath, destPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveDirs removes the list of given directories
func RemoveDirs(dirPath string, dirNames []string) {
	fmt.Println("> Cleaning up...")
	for _, dirName := range dirNames {
		if err := os.RemoveAll(filepath.Join(dirPath, dirName)); err != nil {
			fmt.Println("Warning: Unable to remove .git directory:", err)
		}
	}
}

// EnsureBinary ensures the git binary is present in the system
func EnsureBinary(binary string) {
	// Check if the given binary is installed
	if _, err := exec.LookPath(binary); err != nil {
		log.Fatal(fmt.Sprintf("Error: %s must be installed to use this command.", binary))
	}
}

// HasBinary returns if the given binary is installed in the system
func HasBinary(binary string) bool {
	if _, err := exec.LookPath(binary); err != nil {
		return false
	}
	return true
}

// DirPath returns the full directory path for a directory name
func DirPath(dirname string) string {
	currentDir, _ := os.Getwd()
	return filepath.Join(currentDir, dirname)
}

// EnsureEmptyDir ensures the directory in which the project should be created is empty
func EnsureEmptyDir(dirname string) {
	dirPath := DirPath(dirname)
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// If directory doesn't exist, create it
		fmt.Printf("> Creating new directory: %s\n", dirname)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			log.Fatal("Error creating directory:", err)
		}
	} else {
		// Directory exists, check if it's empty
		files, _ := filepath.Glob(filepath.Join(dirPath, "*"))
		if len(files) > 0 {
			log.Fatal("Error: The directory must be empty to create a new project.")
		}
	}
}

// HasDirectory returns true if a directory exists and false otherwise
func HasDirectory(dirPath string) bool {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// CreateDirIfNotExists creates a directory if it does not exist
func CreateDirIfNotExists(dirPath string) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			log.Fatal("Error creating directory:", err)
		}
	}
}

// RunCommand runs a command in a specific directory
func RunCommand(dirPath string, command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dirPath
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", stdoutStderr)
}

const scaffoldCacheDir = ".cache/lemmego/scaffold"
const scaffoldRepoOwner = "lemmego"
const scaffoldRepoName = "cli"
const scaffoldRepoBranch = "main"

// scaffoldCache returns the path to the local scaffold cache directory.
func scaffoldCache() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, scaffoldCacheDir)
}

// scaffoldCacheVersion returns the cached scaffold version string, or "".
func scaffoldCacheVersion() string {
	cacheDir := scaffoldCache()
	if cacheDir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(cacheDir, "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// fetchLatestScaffold attempts to download the latest scaffold from GitHub main
// into the local cache. Returns true if a newer version was fetched, false if
// the cache is already current or if the fetch fails (non-fatal).
func fetchLatestScaffold() bool {
	cacheDir := scaffoldCache()
	if cacheDir == "" {
		return false
	}

	// Fetch remote VERSION
	versionURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/_scaffold/VERSION",
		scaffoldRepoOwner, scaffoldRepoName, scaffoldRepoBranch)
	resp, err := http.Get(versionURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	remoteVersionBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	remoteVersion := strings.TrimSpace(string(remoteVersionBytes))
	if remoteVersion == "" {
		return false
	}

	// Compare with cached version
	if scaffoldCacheVersion() == remoteVersion && dirExists(filepath.Join(cacheDir, "_scaffold")) {
		return false
	}

	// Download tarball from GitHub
	tarballURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s",
		scaffoldRepoOwner, scaffoldRepoName, scaffoldRepoBranch)

	req, err := http.NewRequest("GET", tarballURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp2.Body.Close()

	// Extract to temp dir, then move _scaffold/ to cache
	tmpDir, err := os.MkdirTemp("", "lemmego-scaffold-*")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tmpDir)

	// Download and extract using system tar
	tarPath, err := os.CreateTemp("", "scaffold-*.tar.gz")
	if err != nil {
		return false
	}
	defer os.Remove(tarPath.Name())

	if _, err := io.Copy(tarPath, resp2.Body); err != nil {
		return false
	}
	tarPath.Close()

	// Extract
	cmd := exec.Command("tar", "-xzf", tarPath.Name(), "-C", tmpDir, "--strip-components=1")
	if err := cmd.Run(); err != nil {
		return false
	}

	// Move _scaffold/ into cache
	os.RemoveAll(filepath.Join(cacheDir, "_scaffold"))
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return false
	}

	if err := CopyDir(filepath.Join(tmpDir, "_scaffold"), filepath.Join(cacheDir, "_scaffold")); err != nil {
		return false
	}

	// Write VERSION
	os.WriteFile(filepath.Join(cacheDir, "VERSION"), []byte(remoteVersion), 0644)

	return true
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// scaffoldDir returns the path to the scaffold to use (cached or embedded).
// Returns the embedded "_scaffold" marker if no cache is available.
func scaffoldDir() string {
	cacheDir := scaffoldCache()
	if cacheDir == "" {
		return ""
	}
	candidate := filepath.Join(cacheDir, "_scaffold")
	if dirExists(candidate) {
		return candidate
	}
	return ""
}
