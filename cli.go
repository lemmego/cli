package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
