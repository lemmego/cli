package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

const repoURL = "https://github.com/lemmego/lemmego"

type Tag struct {
	Name   string `json:"name"`
	Commit struct {
		Sha string `json:"sha"`
	} `json:"commit"`
}

var newCmd = &cobra.Command{
	Use:     "new [dirname]",
	Aliases: []string{"n"},
	Short:   "Create an app",
	Long:    `Create a new Lemmego app`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var newModuleName string
		dirname := args[0]
		dirPath := DirPath(dirname)
		binaries := []string{"go", "node", "git"}

		for _, binary := range binaries {
			EnsureBinary(binary)
		}

		EnsureEmptyDir(dirname)
		ScanStr(&newModuleName, "module")
		createProject(dirPath)
		renameModule(newModuleName, dirPath)
		copyEnvFile(dirPath)
		createSQLiteDatabase(dirPath)
		installNodeModules(dirPath)
		installGoModules(dirPath)
		buildFrontend(dirPath)
		RemoveDirs(dirPath, []string{".git"})

		fmt.Println("\nSuccessfully created a new Lemmego app with module name:", newModuleName, "in directory:", dirname)
		fmt.Println("> Navigate to your new project, and run:")
		fmt.Println("cd", dirname)
		fmt.Println("lemmego run")
	},
}

func npmBinary() string {
	if HasBinary("pnpm") {
		return "pnpm"
	} else if HasBinary("yarn") {
		return "yarn"
	} else if HasBinary("npm") {
		return "npm"
	} else {
		log.Fatal("You need to install pnpm, yarn or npm")
	}
	return ""
}

func installGoModules(dirPath string) {
	fmt.Println("> Installing go modules...")
	RunCommand(dirPath, "go", "mod", "tidy")
	fmt.Println("Done")
}

func installNodeModules(dirPath string) {
	fmt.Println("> Installing node modules...")
	RunCommand(dirPath, npmBinary(), "install")
}

func buildFrontend(dirPath string) {
	fmt.Println("> Building frontend assets...")
	RunCommand(dirPath, npmBinary(), "run", "build")
}

func createSQLiteDatabase(dirPath string) {
	// Create SQLite database file
	storageDir := filepath.Join(dirPath, "storage")
	databaseFile := filepath.Join(storageDir, "database.sqlite")

	// Create the database file
	dbFile, err := os.Create(databaseFile)
	if err != nil {
		fmt.Println("Warning: Could not create SQLite database file:", err)
	} else {
		dbFile.Close() // Ensure the file is closed after creation
		fmt.Println("> Creating SQLite db in ./storage/database.sqlite")
	}
}

func copyEnvFile(dirPath string) {
	// Copy .env.example to .env
	sourceFile := filepath.Join(dirPath, ".env.example")
	destFile := filepath.Join(dirPath, ".env")
	if _, err := os.Stat(sourceFile); err == nil {
		fmt.Println("> Copying .env.example to .env...")
		if err := CopyFile(sourceFile, destFile); err != nil {
			fmt.Println("Warning: Could not copy .env.example to .env:", err)
		}
	} else if !os.IsNotExist(err) {
		fmt.Println("Warning: Error checking for .env.example file:", err)
	}
}

func renameModule(newModuleName string, dirPath string) {
	fmt.Println("> Creating a new project, please wait...")
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".templ") || filepath.Base(path) == "go.mod") {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			newContent := strings.ReplaceAll(string(content), "github.com/lemmego/lemmego", newModuleName)
			if err = os.WriteFile(path, []byte(newContent), 0644); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal("Error replacing module name:", err)
	}
}

func downloadRepo(dirPath string) {
	fmt.Println("> Downloading starter template...")
	_, err := git.PlainClone(dirPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		log.Fatal("Error cloning repository:", err)
	}
}

func createProject(dirPath string) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal("Error getting cache dir:", err)
	}

	commitHash, err := getLatestCommitHash("lemmego", "lemmego")
	cachePath := filepath.Join(cacheDir, "lemmego", commitHash)

	if err != nil {
		log.Fatal("Error getting latest commit hash:", err)
	}

	if HasDirectory(cachePath) {
		err := CopyDir(cachePath, dirPath)
		if err != nil {
			log.Fatal("Error copying lemmego directory from cache:", err)
		}
	} else {
		fmt.Println(fmt.Sprintf("> Caching repo in %s...", cachePath))
		downloadRepo(cachePath)
		err := CopyDir(cachePath, dirPath)
		if err != nil {
			log.Fatal("Error copying lemmego directory from cache:", err)
		}
	}
}

func getLatestCommitHash(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, "main")
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var commit struct {
		Sha string `json:"sha"`
	}
	err = json.Unmarshal(body, &commit)
	if err != nil {
		return "", err
	}

	return commit.Sha, nil
}

func getLatestTag(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tags []Tag
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return "", err
	}

	if len(tags) > 0 {
		return tags[0].Name, nil
	}
	return "", fmt.Errorf("No tags found")
}
