module {{.ModuleName}}

go 1.27

require (
	{{- if .HasTempl}}
	github.com/a-h/templ {{.Version "github.com/a-h/templ"}}
	{{- end}}
	github.com/spf13/cobra {{.Version "github.com/spf13/cobra"}}
)

require (
	github.com/lemmego/api {{.Version "github.com/lemmego/api"}}
	github.com/lemmego/migration {{.Version "github.com/lemmego/migration"}}
	{{- if .HasCache}}
	github.com/lemmego/cache {{.Version "github.com/lemmego/cache"}}
	{{- end}}
	{{- if .HasQueue}}
	github.com/lemmego/queue {{.Version "github.com/lemmego/queue"}}
	{{- end}}
	{{- if .EnableAuth}}
	github.com/lemmego/auth {{.Version "github.com/lemmego/auth"}}
	{{- end}}
	{{- if .UseGPA}}
	github.com/lemmego/gpa {{.Version "github.com/lemmego/gpa"}}
	{{- end}}
	{{- if .UseORMConnector}}
	github.com/lemmego/orm {{.Version "github.com/lemmego/orm"}}
	github.com/lemmego/ormconnector {{.Version "github.com/lemmego/ormconnector"}}
	{{- if .UseGPA}}
	github.com/lemmego/gpaorm {{.Version "github.com/lemmego/gpaorm"}}
	{{- end}}
	{{- end}}
	{{- if .UseGormConnector}}
	github.com/lemmego/gormconnector {{.Version "github.com/lemmego/gormconnector"}}
	{{- if .UseGPA}}
	github.com/lemmego/gpagorm {{.Version "github.com/lemmego/gpagorm"}}
	{{- end}}
	{{- end}}
	{{- if .UseBunConnector}}
	github.com/lemmego/bunconnector {{.Version "github.com/lemmego/bunconnector"}}
	{{- if .UseGPA}}
	github.com/lemmego/gpabun {{.Version "github.com/lemmego/gpabun"}}
	{{- end}}
	{{- end}}
	{{- if .InertiaProvider}}
	github.com/lemmego/inertia {{.Version "github.com/lemmego/inertia"}}
	{{- end}}
)
