package cli

import (
	"bytes"
	"errors"
	"log"
	"reflect"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/spf13/cobra"
)

var UiDataTypeMap = map[string]string{
	"text":     reflect.String.String(),
	"textarea": reflect.String.String(),
	"integer":  reflect.Uint.String(),
	"decimal":  reflect.Float64.String(),
	"boolean":  reflect.Bool.String(),
	"radio":    reflect.String.String(),
	"checkbox": reflect.String.String(),
	"dropdown": reflect.String.String(),
	"date":     "time.Time",
	"time":     "time.Time",
	"file":     reflect.String.String(),
}

var UiDbTypeMap = map[string]string{
	"text":     "string",
	"textarea": "text",
	"integer":  "unsignedBigInt",
	"decimal":  "decimal",
	"boolean":  "boolean",
	"radio":    "string",
	"checkbox": "string",
	"dropdown": "string",
	"date":     "dateTime",
	"time":     "time",
	"file":     "string",
}

var CommonFuncs = template.FuncMap{
	"contains":  strings.Contains,
	"hasSuffix": strings.HasSuffix,
	"join":      strings.Join,

	"toTitle": func(str string) string {
		caser := cases.Title(language.English)
		return caser.String(str)
	},
	"toCamel": func(str string) string {
		strcase.ConfigureAcronym("id", "ID")
		return strcase.ToCamel(str)
	},
	"toLowerCamel": func(str string) string {
		strcase.ConfigureAcronym("id", "ID")
		return strcase.ToLowerCamel(str)
	},
	"toSnake": func(str string) string {
		return strcase.ToSnake(str)
	},
	"toSpaceDelimited": func(str string) string {
		return strcase.ToDelimited(str, ' ')
	},
	"concat": func(str string, strs ...string) string {
		return str + strings.Join(strs, "")
	},
}

type Gen struct {
	Name       string
	Appendable []byte
}

type Replacable struct {
	Placeholder string
	Value       interface{}
}

type Generator interface {
	Generate() error
}

type PackagePathGetter interface {
	GetPackagePath() string
}

type StubGetter interface {
	GetStub() string
}

type Commander interface {
	Command() *cobra.Command
}

type CommandGenerator interface {
	Generator
	PackagePathGetter
	StubGetter
	Commander
}

// ParseTemplate renders a generator stub.
//
// text/template, not html/template, because the output is Go and TypeScript
// source rather than a web page. html/template escapes every interpolated
// value for HTML and — worse — reads the literal template text as HTML to
// decide the escaping context. Literal markup such as <div> survived that, so
// the JSX stubs looked fine, but `useForm<{{.Name}}Fields>` did not: an action
// where a tag name belongs is not something the HTML parser will emit, so it
// rewrote the "<" to "&lt;" and the generated form could not compile.
//
// The giveaway that this was already hurting was the line this replaced — a
// strings.ReplaceAll turning "&#34;" back into a quote. It patched the one
// entity somebody hit; "<", ">", "&" and "'" in an interpolated value were
// still being mangled, silently, with no second hack to catch them.
func ParseTemplate(tmplData map[string]interface{}, fileContents string, funcMap template.FuncMap) (string, error) {
	var out bytes.Buffer
	// missingkey=error, because the alternative is silence. Appendable was
	// set only when it had content, so every generated form and handler ended
	// with the literal "<no value>" — valid template output, invalid Go and
	// invalid TSX, and nothing said so. A generator that cannot fill a
	// placeholder should refuse rather than write the placeholder's error
	// message into somebody's source file.
	tx := template.New("template").Option("missingkey=error")
	if funcMap != nil {
		tx.Funcs(funcMap)
	}
	t := template.Must(tx.Parse(fileContents))
	err := t.Execute(&out, tmplData)
	if err != nil {
		return "", errors.New("Unable to execute template:" + err.Error())
	}
	return out.String(), nil
}

// genCmd represents the generator command
var genCmd = &cobra.Command{
	Use:     "gen",
	Aliases: []string{"g"},
	Short:   "Generate code",
	Long:    `Generate code`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("An argument must be provided to the gen command (e.g. model, input, migration, handlers, etc.)")
	},
}
