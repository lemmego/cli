package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEnvironmentFinalizationHelpers(t *testing.T) {
	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("# APP_KEY=leave this comment\nAPP_KEY=old\nNAME=value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := setEnvValue(envPath, "APP_KEY", "new"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(body); got != "# APP_KEY=leave this comment\nAPP_KEY=new\nNAME=value\n" {
		t.Fatalf("updated env = %q", got)
	}

	if err := setEnvValue(envPath, "SECRET", "value"); err != nil {
		t.Fatal(err)
	}
	body, _ = os.ReadFile(envPath)
	if !strings.HasSuffix(string(body), "SECRET=value\n") {
		t.Fatalf("missing appended env value: %q", body)
	}

	example := filepath.Join(root, ".env.example")
	if err := os.WriteFile(example, []byte("APP_NAME=demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	copyEnvFile(root)
	if got, err := os.ReadFile(filepath.Join(root, ".env")); err != nil || string(got) != "APP_NAME=demo\n" {
		t.Fatalf("copied env = %q, error = %v", got, err)
	}

	generateAppKey(root)
	body, err = os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	line := ""
	for _, candidate := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(candidate, "APP_KEY=") {
			line = candidate
		}
	}
	if len(line) != len("APP_KEY=")+44 {
		t.Fatalf("generated APP_KEY has unexpected length: %q", line)
	}
}

func TestRenameModuleUpdatesSupportedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/lemmego/lemmego\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nimport \"github.com/lemmego/lemmego/api\"\n\nfunc main() { _ = api.Context{} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("github.com/lemmego/lemmego\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	renameModule("github.com/example/app", root)
	for _, name := range []string{"go.mod", "main.go"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "github.com/example/app") || strings.Contains(string(body), "github.com/lemmego/lemmego") {
			t.Fatalf("%s was not renamed: %s", name, body)
		}
	}
	body, _ := os.ReadFile(filepath.Join(root, "notes.txt"))
	if !strings.Contains(string(body), "github.com/lemmego/lemmego") {
		t.Fatal("renameModule changed an unrelated file")
	}
	if err := renameModule("github.com/example/app", filepath.Join(root, "missing")); err == nil {
		t.Fatal("renameModule accepted a missing directory")
	}
}

func TestPureCommandDecisions(t *testing.T) {
	for _, test := range []struct {
		args []string
		want bool
	}{
		{nil, true},
		{[]string{"--verbose", "serve"}, true},
		{[]string{"migrate", "up"}, false},
		{[]string{"rollback"}, false},
		{[]string{"key"}, false},
		{[]string{"cache:clear"}, false},
		{[]string{"serve"}, true},
	} {
		if got := needsFrontend(test.args); got != test.want {
			t.Errorf("needsFrontend(%v) = %v, want %v", test.args, got, test.want)
		}
	}

	root := t.TempDir()
	chdirForTest(t, root)
	if err := os.MkdirAll(filepath.Join(root, "resources", "js"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "js", "ssr.tsx"), []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasInertiaSSREntry() {
		t.Fatal("expected SSR entry")
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"build:ssr":"vite build"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasNpmScript("build:ssr") || hasNpmScript("missing") {
		t.Fatal("unexpected npm script detection")
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if hasNpmScript("build:ssr") {
		t.Fatal("invalid package.json reported a script")
	}
	if err := os.Remove(filepath.Join(root, "resources", "js", "ssr.tsx")); err != nil {
		t.Fatal(err)
	}
	if hasInertiaSSREntry() {
		t.Fatal("missing SSR entry was reported")
	}

	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "page.templ"), []byte("templ page() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasTemplFiles() {
		t.Fatal("expected templ file")
	}
	if err := os.WriteFile(filepath.Join(root, "cmd.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isLemmegoProject() {
		t.Fatal("non-Lemmego directory was detected as a project")
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "app", "main.go"), []byte("package main\n\nimport \"github.com/lemmego/api/app\"\n\nvar _ app.Context\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !isLemmegoProject() {
		t.Fatal("Lemmego directory was not detected")
	}
	if got := processPrefix("unknown"); !strings.Contains(got, "[unknown]") {
		t.Fatalf("processPrefix() = %q", got)
	}
}

func TestProjectConfigurationValidationMatrix(t *testing.T) {
	withFlags(t, projectFlagSet{Module: "github.com/example/app", Redis: true, Auth: false})
	cfg, err := collectNonInteractiveProjectConfig("app")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheDriver != CacheRedis || cfg.QueueDriver != QueueRedis || cfg.SessionDriver != SessionRedis {
		t.Fatalf("--redis did not set unset drivers: %+v", cfg)
	}

	withFlags(t, projectFlagSet{Module: "github.com/example/app", Redis: true, Cache: "memory", Auth: false})
	cfg, err = collectNonInteractiveProjectConfig("app")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheDriver != CacheMemory || cfg.QueueDriver != QueueRedis {
		t.Fatalf("explicit cache did not override --redis: %+v", cfg)
	}

	for field, set := range map[string]func(*projectFlagSet){
		"preset":   func(f *projectFlagSet) { f.Preset = "invalid" },
		"frontend": func(f *projectFlagSet) { f.Frontend = "invalid" },
		"database": func(f *projectFlagSet) { f.Database = "invalid" },
		"orm":      func(f *projectFlagSet) { f.ORM = "invalid" },
		"cache":    func(f *projectFlagSet) { f.Cache = "invalid" },
		"queue":    func(f *projectFlagSet) { f.Queue = "invalid" },
		"session":  func(f *projectFlagSet) { f.Session = "invalid" },
		"disk":     func(f *projectFlagSet) { f.Disk = "invalid" },
	} {
		flags := projectFlagSet{Module: "github.com/example/app", Auth: false}
		set(&flags)
		withFlags(t, flags)
		if _, err := collectNonInteractiveProjectConfig("app"); err == nil {
			t.Errorf("invalid %s was accepted", field)
		}
	}

	withFlags(t, projectFlagSet{Module: "github.com/example/app", Database: "none", Queue: "none", Auth: true})
	if _, err := collectNonInteractiveProjectConfig("app"); err == nil || !strings.Contains(err.Error(), "authentication needs") {
		t.Fatalf("expected auth/database validation error, got %v", err)
	}

	withFlags(t, projectFlagSet{Module: "github.com/example/app", Database: "none", Auth: false, Queue: "sql"})
	if _, err := collectNonInteractiveProjectConfig("app"); err == nil || !strings.Contains(err.Error(), "SQL queue") {
		t.Fatalf("expected SQL queue validation error, got %v", err)
	}

	withFlags(t, projectFlagSet{Module: "github.com/example/app", Database: "none", Auth: false, GPA: true, Queue: "none"})
	cfg, err = collectNonInteractiveProjectConfig("app")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ORM != OrmNone || cfg.EnableGPA {
		t.Fatalf("database none did not disable ORM/GPA: %+v", cfg)
	}
}

func TestGetModuleNameAndFilesystemHelpers(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	if err := os.WriteFile("go.mod", []byte("module github.com/example/app\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := GetModuleName(); err != nil || got != "github.com/example/app" {
		t.Fatalf("GetModuleName() = %q, %v", got, err)
	}
	if err := os.WriteFile("go.mod", []byte("go 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := GetModuleName(); err == nil {
		t.Fatal("expected missing module declaration error")
	}

	src := filepath.Join(root, "src.txt")
	dst := filepath.Join(root, "nested", "dst.txt")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err == nil {
		t.Fatal("expected CopyFile to reject missing destination directory")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	if !HasDirectory(filepath.Dir(dst)) || !HasBinary("go") {
		t.Fatal("filesystem helper returned unexpected result")
	}

	if err := os.MkdirAll(filepath.Join(root, "tree", "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tree", "child", "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyDir(filepath.Join(root, "tree"), filepath.Join(root, "copy")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "copy", "child", "file")); err != nil {
		t.Fatal(err)
	}
}

func TestBuildCommandRunsExpectedSteps(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(filepath.Join(root, "cmd", "app", "resources", "js"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "app", "main.go"), []byte("package main\n\nimport \"github.com/lemmego/api/app\"\n\nvar _ app.Context\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "js"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "js", "ssr.tsx"), []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "templates", "page.templ"), []byte("templ page() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"build":"vite build","build:ssr":"vite build --ssr"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, viteHotFile), []byte("http://localhost:5173"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "commands.log")
	script := "#!/bin/sh\nprintf '%s %s\\n' \"$0\" \"$*\" >> \"" + logPath + "\"\n"
	for _, name := range []string{"templ", "npm", "pnpm", "yarn"} {
		path := filepath.Join(bin, name)
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	buildCmd.Run(buildCmd, nil)
	if _, err := os.Stat(viteHotFile); !os.IsNotExist(err) {
		t.Fatalf("build did not remove %s", viteHotFile)
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"templ generate", "npm run build", "npm run build:ssr"} {
		if !strings.Contains(string(log), want) {
			t.Errorf("build did not run %q; log=%s", want, log)
		}
	}
}

func TestRunCommandForwardsApplicationArguments(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	if err := os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "app", "main.go"), []byte("package main\n\nimport \"github.com/lemmego/api/app\"\n\nvar _ app.Context\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	previous := runApplication
	t.Cleanup(func() { runApplication = previous })
	var got []string
	runApplication = func(args []string) error {
		got = append([]string(nil), args...)
		return nil
	}
	runCmd.Run(runCmd, []string{"migrate", "--database", "test"})
	if strings.Join(got, " ") != "migrate --database test" {
		t.Fatalf("run command forwarded %v", got)
	}
}

func TestDestinationValidationAndDirectoryHelpers(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)

	if _, err := validateDestination(""); err == nil {
		t.Fatal("empty destination was accepted")
	}
	if _, err := validateDestination("../outside"); err == nil {
		t.Fatal("escaping destination was accepted")
	}
	if _, err := validateDestination(filepath.Join(root, "absolute")); err == nil {
		t.Fatal("absolute destination was accepted")
	}
	if err := ensureEmptyDir("../outside"); err == nil {
		t.Fatal("ensureEmptyDir accepted an invalid destination")
	}

	created, err := validateDestinationContents(filepath.Join(root, "new", "app"))
	if err != nil || !created {
		t.Fatalf("new destination = %v, %v", created, err)
	}
	if created, err := validateDestinationContents(filepath.Join(root, "new", "app")); err != nil || created {
		t.Fatalf("empty existing destination = %v, %v", created, err)
	}
	if err := os.WriteFile(filepath.Join(root, "new", "app", "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := validateDestinationContents(filepath.Join(root, "new", "app")); err == nil {
		t.Fatal("non-empty destination was accepted")
	}
	if _, err := validateDestinationContents(filepath.Join(root, "new", "app", "file")); err == nil {
		t.Fatal("file destination was accepted")
	}

	if err := os.MkdirAll(filepath.Join(root, "nested", "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if HasDirectory(filepath.Join(root, "nested", "dir")) == false || HasDirectory(filepath.Join(root, "missing")) {
		t.Fatal("HasDirectory returned an unexpected result")
	}
	CreateDirIfNotExists(filepath.Join(root, "created", "dir"))
	if !HasDirectory(filepath.Join(root, "created", "dir")) {
		t.Fatal("CreateDirIfNotExists did not create the directory")
	}
	RemoveDirs(root, []string{"created", "nested"})
	if HasDirectory(filepath.Join(root, "created")) || HasDirectory(filepath.Join(root, "nested")) {
		t.Fatal("RemoveDirs did not remove requested directories")
	}
}

func TestSSRProcessAndCacheHelpers(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	if _, err := readPidFile(); err == nil {
		t.Fatal("missing PID file was accepted")
	}
	if err := os.WriteFile(ssrPidFile, []byte("not-a-pid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPidFile(); err == nil {
		t.Fatal("malformed PID file was accepted")
	}
	if err := os.WriteFile(ssrPidFile, []byte("  "+strconv.Itoa(os.Getpid())+"  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pid, err := readPidFile()
	if err != nil || pid != os.Getpid() || !processRunning(pid) {
		t.Fatalf("readPidFile/processRunning = %d, %v", pid, err)
	}
	if processRunning(-1) {
		t.Fatal("negative PID reported as running")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheDir := scaffoldCache()
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "VERSION"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheCleanCmd.Run(cacheCleanCmd, nil)
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("cache directory still exists: %v", err)
	}
	cacheCleanCmd.Run(cacheCleanCmd, nil)
}

func TestExecuteRegistersCommandsOnce(t *testing.T) {
	previousArgs := rootCmd.Flags().Args()
	previousOut := rootCmd.OutOrStdout()
	t.Cleanup(func() {
		rootCmd.SetArgs(previousArgs)
		rootCmd.SetOut(previousOut)
	})
	rootCmd.SetArgs([]string{"--version"})
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	if err := Execute(); err != nil {
		t.Fatal(err)
	}
	if err := Execute(); err != nil {
		t.Fatal(err)
	}
	if rootCmd.Commands() == nil {
		t.Fatal("root command did not register subcommands")
	}
}

func TestResolveProjectPathsUsesProjectOutputAndFallbacks(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	if err := os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "app", "main.go"), []byte("package main\n\nimport \"github.com/lemmego/api/app\"\n\nvar _ app.Context\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	goPath := filepath.Join(bin, "go")
	if err := os.WriteFile(goPath, []byte("#!/bin/sh\nprintf 'boot log\\n{\"model_path\":\"./domain/models\",\"handler_path\":\"./domain/handlers\"}\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	paths := resolveProjectPaths()
	if paths.ModelPath != "./domain/models" || paths.HandlerPath != "./domain/handlers" || paths.InputPath != "./internal/inputs" {
		t.Fatalf("resolved paths = %+v", paths)
	}

	if err := os.WriteFile(goPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	paths = resolveProjectPaths()
	if paths != conventionalPaths() {
		t.Fatalf("failed command did not use conventional paths: %+v", paths)
	}
}

func TestDevProcessLifecycleHelpers(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	for _, name := range []string{"air", "templ", "vite", "other"} {
		if got := processPrefix(name); !strings.Contains(got, "["+name+"]") {
			t.Errorf("processPrefix(%q) = %q", name, got)
		}
	}

	if err := os.MkdirAll(filepath.Join(root, "deep", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "deep", "nested", "page.templ"), []byte("templ page() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasTemplFiles() {
		t.Fatal("hasTemplFiles missed a nested template")
	}

	process := startDevProcess("test", root, "sh", "-c", "sleep 10")
	if process.cmd.Process == nil {
		t.Fatal("startDevProcess did not start a process")
	}
	killAll([]devProcess{process})
}

func TestWaitForViteHotFileWaitsForFreshFile(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	if err := os.MkdirAll(filepath.Dir(viteHotPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(viteHotPath, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(25 * time.Millisecond)
		_ = os.WriteFile(viteHotPath, []byte("fresh"), 0o644)
	}()
	waitForViteHotFile()
	if body, err := os.ReadFile(viteHotPath); err != nil || string(body) != "fresh" {
		t.Fatalf("hot file = %q, %v", body, err)
	}
}

func TestNewCommandOrchestratesProjectCreation(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	t.Setenv("LEMMEGO_SCAFFOLD_SOURCE", "embedded")

	previousFlags := projectFlags
	previousNonInteractive := nonInteractive
	previousRunner := runCommand
	t.Cleanup(func() {
		projectFlags = previousFlags
		nonInteractive = previousNonInteractive
		runCommand = previousRunner
	})
	projectFlags = projectFlagSet{
		Module:   "github.com/example/generated-app",
		Preset:   "mvc",
		Frontend: "go_templates",
		ORM:      "orm",
		Database: "sqlite",
		Cache:    "memory",
		Queue:    "none",
		Session:  "memory",
		Disk:     "local",
		Auth:     false,
	}
	nonInteractive = true
	var commands []string
	runCommand = func(dirPath string, command string, args ...string) ([]byte, error) {
		commands = append(commands, command+" "+strings.Join(args, " "))
		return nil, nil
	}

	if err := newCmd.RunE(newCmd, []string{"generated-app"}); err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0] != "go mod tidy" {
		t.Fatalf("external commands = %v", commands)
	}
	for _, path := range []string{
		"generated-app/go.mod",
		"generated-app/.env",
		"generated-app/storage/database.sqlite",
		"generated-app/internal/configs/app.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("generated project missing %s: %v", path, err)
		}
	}
	module, err := os.ReadFile("generated-app/go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(module), "module github.com/example/generated-app") {
		t.Fatalf("module was not renamed: %s", module)
	}

	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"node", "npm", "pnpm"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	commands = nil
	projectFlags = projectFlagSet{
		Module:   "github.com/example/generated-react",
		Preset:   "mvc",
		Frontend: "inertia_react",
		ORM:      "orm",
		Database: "sqlite",
		Cache:    "memory",
		Queue:    "none",
		Session:  "memory",
		Disk:     "local",
		Auth:     false,
	}
	if err := newCmd.RunE(newCmd, []string{"generated-react"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pnpm install", "pnpm run build", "go mod tidy"} {
		found := false
		for _, command := range commands {
			if command == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("frontend project did not run %q: %v", want, commands)
		}
	}
}
