package cli

import (
	_ "embed"
	"fmt"

	"github.com/charmbracelet/huh"

	"strings"

	"github.com/lemmego/fsys"

	"github.com/spf13/cobra"
)

//go:embed handler.txt
var handlerStub string

type HandlerField struct {
	Name string
}

type HandlerConfig struct {
	Name string
}

type HandlerGenerator struct {
	name string
}

func NewHandlerGenerator(mc *HandlerConfig) *HandlerGenerator {
	return &HandlerGenerator{mc.Name}
}

func (hg *HandlerGenerator) GetPackagePath() string {
	return "internal/handlers"
}

func (hg *HandlerGenerator) GetStub() string {
	return handlerStub
}

func (hg *HandlerGenerator) Generate(appendable ...[]byte) error {
	fs := fsys.NewLocalStorage("")
	parts := strings.Split(hg.GetPackagePath(), "/")
	packageName := hg.GetPackagePath()

	if len(parts) > 0 {
		packageName = parts[len(parts)-1]
	}

	tmplData := map[string]interface{}{
		"PackageName": packageName,
		"Name":        hg.name,
	}

	if len(appendable) > 0 {
		tmplData["Appendable"] = string(appendable[0])
	}

	output, err := ParseTemplate(tmplData, hg.GetStub(), commonFuncs)

	if err != nil {
		return err
	}

	err = fs.Write(hg.GetPackagePath()+"/"+hg.name+"_handlers.go", []byte(output))

	if err != nil {
		return err
	}

	return nil
}

func (hg *HandlerGenerator) Command() *cobra.Command {
	return handlerCmd
}

var handlerCmd = &cobra.Command{
	Use:     "handlers",
	Aliases: []string{"h"},
	Short:   "Generate a handler set",
	Long:    `Generate a handler set`,
	Run: func(cmd *cobra.Command, args []string) {
		var handlerName string

		if !shouldRunInteractively && len(args) == 0 {
			fmt.Println("Please provide a handler name")
			return
		}

		if shouldRunInteractively && len(args) == 0 {
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter the resource name in snake_case").
						Value(&handlerName).
						Validate(SnakeCase),
				),
			)

			err := form.Run()

			if err != nil {
				fmt.Println(err)
				return
			}
		} else {
			handlerName = args[0]
		}

		mg := NewHandlerGenerator(&HandlerConfig{Name: handlerName})
		err := mg.Generate()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Handler generated successfully.")
	},
}
