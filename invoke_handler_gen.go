package cli

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/lemmego/fsys"
	"github.com/spf13/cobra"
)

//go:embed invoke_handler.txt
var invokeHandlerStub string

type InvokeHandlerConfig struct {
	Name string
}

type InvokeHandlerGenerator struct {
	name string
}

func NewInvokeHandlerGenerator(config *InvokeHandlerConfig) *InvokeHandlerGenerator {
	return &InvokeHandlerGenerator{name: config.Name}
}

func (g *InvokeHandlerGenerator) GetPackagePath() string {
	return Paths().HandlerPath
}

func (g *InvokeHandlerGenerator) Generate() error {
	fs := fsys.NewLocalStorage("")
	parts := strings.Split(g.GetPackagePath(), "/")
	packageName := parts[len(parts)-1]
	output, err := ParseTemplate(map[string]interface{}{
		"PackageName": packageName,
		"Name":        g.name,
	}, invokeHandlerStub, CommonFuncs)
	if err != nil {
		return err
	}
	if err := ensurePackageDir(fs, g.GetPackagePath()); err != nil {
		return err
	}
	return fs.Write(g.GetPackagePath()+"/"+g.name+"_handler.go", []byte(output))
}

func (g *InvokeHandlerGenerator) Command() *cobra.Command {
	return invokeHandlerCmd
}

var invokeHandlerCmd = &cobra.Command{
	Use:   "invoke <name>",
	Short: "Generate one non-resourceful handler",
	Long:  "Generate a handler with a single InvokeHandler method. Use a snake_case name such as post_export.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if err := SnakeCase(name); err != nil {
			return fmt.Errorf("name must be snake_case: %w", err)
		}
		if err := NewInvokeHandlerGenerator(&InvokeHandlerConfig{Name: name}).Generate(); err != nil {
			return err
		}
		fmt.Println("Invokable handler generated successfully.")
		return nil
	},
}
