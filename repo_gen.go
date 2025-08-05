package cli

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/lemmego/fsys"
	"github.com/spf13/cobra"
)

//go:embed repo.txt
var repoStub string

type RepoConfig struct {
	Name string
}

type RepoGenerator struct {
	name string
}

func NewRepoGenerator(rc *RepoConfig) *RepoGenerator {
	return &RepoGenerator{rc.Name}
}

func (rg *RepoGenerator) GetPackagePath() string {
	return "internal/repos"
}

func (rg *RepoGenerator) GetStub() string {
	return repoStub
}

func (rg *RepoGenerator) Generate(appendable ...[]byte) error {
	fs := fsys.NewLocalStorage("")
	parts := strings.Split(rg.GetPackagePath(), "/")
	packageName := rg.GetPackagePath()

	if len(parts) > 0 {
		packageName = parts[len(parts)-1]
	}

	modName, err := getModuleName()
	if err != nil {
		return err
	}

	tmplData := map[string]interface{}{
		"PackageName": packageName,
		"Name":        rg.name,
		"ModuleName":  modName,
	}

	if len(appendable) > 0 {
		tmplData["Appendable"] = string(appendable[0])
	}

	output, err := ParseTemplate(tmplData, rg.GetStub(), commonFuncs)

	if err != nil {
		return err
	}

	if exists, _ := fs.Exists(rg.GetPackagePath()); exists {
		err = fs.Write(rg.GetPackagePath()+"/"+rg.name+"_repo.go", []byte(output))

		if err != nil {
			return err
		}
	} else {
		fs.CreateDirectory(rg.GetPackagePath())
		err = fs.Write(rg.GetPackagePath()+"/"+rg.name+"_repo.go", []byte(output))

		if err != nil {
			return err
		}
	}

	return nil
}

func (rg *RepoGenerator) Command() *cobra.Command {
	return repoCmd
}

var repoCmd = &cobra.Command{
	Use:     "repo",
	Aliases: []string{"r"},
	Short:   "Generate a repository",
	Long:    `Generate a repository with embedded GPA repository and custom methods`,
	Run: func(cmd *cobra.Command, args []string) {
		var repoName string

		if !shouldRunInteractively && len(args) == 0 {
			fmt.Println("Please provide a repository name")
			return
		}

		if shouldRunInteractively && len(args) == 0 {
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter the repository name in snake_case (e.g. 'user' for UserRepository)").
						Value(&repoName).
						Validate(SnakeCase),
				),
			)

			err := form.Run()

			if err != nil {
				fmt.Println(err)
				return
			}
		} else {
			repoName = args[0]
		}

		rg := NewRepoGenerator(&RepoConfig{Name: repoName})
		err := rg.Generate()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Repository generated successfully.")
	},
}
