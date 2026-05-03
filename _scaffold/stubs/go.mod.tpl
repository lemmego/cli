module {{.ModuleName}}

go 1.24.3

require (
	{{- if .HasTempl}}
	github.com/a-h/templ {{.Version "github.com/a-h/templ"}}
	{{- end}}
	github.com/spf13/cobra {{.Version "github.com/spf13/cobra"}}
)

require (
	github.com/lemmego/api {{.Version "github.com/lemmego/api"}}
	{{- if .EnableAuth}}
	github.com/lemmego/auth {{.Version "github.com/lemmego/auth"}}
	{{- end}}
	{{- if eq .ORM "gorm"}}
	github.com/lemmego/gormconnector {{.Version "github.com/lemmego/gormconnector"}}
	{{- end}}
	{{- if eq .ORM "bun"}}
	github.com/lemmego/bunconnector {{.Version "github.com/lemmego/bunconnector"}}
	{{- end}}
	github.com/lemmego/gpa {{.Version "github.com/lemmego/gpa"}}
	{{- if eq .ORM "gorm"}}
	github.com/lemmego/gpagorm {{.Version "github.com/lemmego/gpagorm"}}
	{{- end}}
	{{- if eq .ORM "bun"}}
	github.com/lemmego/gpabun {{.Version "github.com/lemmego/gpabun"}}
	{{- end}}
	{{- if .InertiaProvider}}
	github.com/lemmego/inertia {{.Version "github.com/lemmego/inertia"}}
	{{- end}}
	github.com/lemmego/migration {{.Version "github.com/lemmego/migration"}}
)
