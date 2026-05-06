package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

type ProjectPreset string

const (
	PresetMVC     ProjectPreset = "mvc"
	PresetRESTAPI ProjectPreset = "rest_api"
)

type OrmChoice string

const (
	OrmGORM OrmChoice = "gorm"
	OrmBun  OrmChoice = "bun"
)

type FrontendPreset string

const (
	FrontendGoTemplates       FrontendPreset = "go_templates"
	FrontendTempl             FrontendPreset = "templ"
	FrontendInertiaReact      FrontendPreset = "inertia_react"
	FrontendInertiaVue        FrontendPreset = "inertia_vue"
	FrontendTemplInertiaReact FrontendPreset = "templ_inertia_react"
	FrontendTemplInertiaVue   FrontendPreset = "templ_inertia_vue"
)

func (f FrontendPreset) HasInertia() bool {
	return f == FrontendInertiaReact || f == FrontendInertiaVue ||
		f == FrontendTemplInertiaReact || f == FrontendTemplInertiaVue
}

func (f FrontendPreset) HasTempl() bool {
	return f == FrontendTempl || f == FrontendTemplInertiaReact || f == FrontendTemplInertiaVue
}

func (f FrontendPreset) HasNodeDeps() bool {
	return f.HasInertia()
}

type ProjectConfig struct {
	Name        string
	ModuleName  string
	Preset      ProjectPreset
	ORM         OrmChoice
	EnableRedis bool
	EnableAuth  bool
	EnableGPA   bool
	Frontend    FrontendPreset
}

func collectProjectConfig(dirname string, enableExperimental bool) *ProjectConfig {
	cfg := ProjectConfig{Name: dirname}

	var moduleName string
	var preset string
	var orm string
	var enableRedis bool
	var enableAuth bool
	var enableGPA bool

	formFields := []huh.Field{
		huh.NewInput().
			Title("Module Name (e.g. github.com/username/repo)").
			Value(&moduleName).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("module name is required")
				}
				return nil
			}),
		huh.NewSelect[string]().
			Title("Preset").
			Options(
				huh.NewOption("MVC", "mvc"),
				huh.NewOption("REST API", "rest_api"),
			).
			Value(&preset),
		huh.NewSelect[string]().
			Title("Choose an SQL ORM").
			Options(
				huh.NewOption("GORM", "gorm"),
				huh.NewOption("Bun", "bun"),
			).
			Value(&orm),
		huh.NewConfirm().
			Title("Enable Redis?").
			Affirmative("Yes").
			Negative("No").
			Value(&enableRedis),
		huh.NewConfirm().
			Title("Enable Auth?").
			Affirmative("Yes").
			Negative("No").
			Value(&enableAuth),
	}

	if enableExperimental {
		formFields = append(formFields,
			huh.NewConfirm().
				Title("Enable GPA? (experimental)").
				Affirmative("Yes").
				Negative("No").
				Value(&enableGPA),
		)
	}

	form1 := huh.NewForm(
		huh.NewGroup(formFields...),
	)

	if err := form1.Run(); err != nil {
		return nil
	}

	cfg.ModuleName = moduleName
	cfg.Preset = ProjectPreset(preset)
	cfg.ORM = OrmChoice(orm)
	cfg.EnableRedis = enableRedis
	cfg.EnableAuth = enableAuth
	cfg.EnableGPA = enableGPA

	if cfg.Preset == PresetMVC {
		var frontend string
		form2 := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose a frontend preset").
					Options(
						huh.NewOption("Go Templates", "go_templates"),
						huh.NewOption("Templ (Go Templates included)", "templ"),
						huh.NewOption("Inertia (React)", "inertia_react"),
						huh.NewOption("Inertia (Vue)", "inertia_vue"),
						huh.NewOption("Templ + Inertia (React)", "templ_inertia_react"),
						huh.NewOption("Templ + Inertia (Vue)", "templ_inertia_vue"),
					).
					Value(&frontend),
			),
		)
		if err := form2.Run(); err != nil {
			return nil
		}
		cfg.Frontend = FrontendPreset(frontend)
	}

	return &cfg
}
