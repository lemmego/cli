package cli

import (
	"html/template"
	"strings"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	data := map[string]interface{}{
		"Name": "User",
	}
	result, err := ParseTemplate(data, "Hello {{.Name}}!", template.FuncMap{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello User!" {
		t.Errorf("expected 'Hello User!', got %s", result)
	}
}

func TestParseTemplateWithFuncs(t *testing.T) {
	data := map[string]interface{}{
		"items": []string{"a", "b", "c"},
	}
	funcs := template.FuncMap{
		"join": strings.Join,
	}
	result, err := ParseTemplate(data, "{{join .items \", \"}}", funcs)
	if err != nil {
		t.Fatal(err)
	}
	if result != "a, b, c" {
		t.Errorf("expected 'a, b, c', got %s", result)
	}
}

func TestToTitle(t *testing.T) {
	fn := CommonFuncs["toTitle"].(func(string) string)
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "Hello"},
		{"user", "User"},
		{"hello_world", "Hello_world"},
		{"", ""},
	}
	for _, tt := range tests {
		got := fn(tt.input)
		if got != tt.want {
			t.Errorf("toTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToCamel(t *testing.T) {
	fn := CommonFuncs["toCamel"].(func(string) string)
	tests := []struct {
		input string
		want  string
	}{
		{"hello_world", "HelloWorld"},
		{"user", "User"},
		{"my_name", "MyName"},
		{"", ""},
	}
	for _, tt := range tests {
		got := fn(tt.input)
		if got != tt.want {
			t.Errorf("toCamel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToLowerCamel(t *testing.T) {
	fn := CommonFuncs["toLowerCamel"].(func(string) string)
	tests := []struct {
		input string
		want  string
	}{
		{"hello_world", "helloWorld"},
		{"user", "user"},
		{"my_name", "myName"},
		{"", ""},
	}
	for _, tt := range tests {
		got := fn(tt.input)
		if got != tt.want {
			t.Errorf("toLowerCamel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToSnake(t *testing.T) {
	fn := CommonFuncs["toSnake"].(func(string) string)
	tests := []struct {
		input string
		want  string
	}{
		{"HelloWorld", "hello_world"},
		{"User", "user"},
		{"MyName", "my_name"},
		{"UserProfile", "user_profile"},
		{"", ""},
	}
	for _, tt := range tests {
		got := fn(tt.input)
		if got != tt.want {
			t.Errorf("toSnake(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestUiDataTypeMap(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"text", "string"},
		{"integer", "uint"},
		{"boolean", "bool"},
		{"date", "time.Time"},
	}
	for _, tt := range tests {
		got, ok := UiDataTypeMap[tt.key]
		if !ok {
			t.Errorf("missing key %q in UiDataTypeMap", tt.key)
			continue
		}
		if got != tt.want {
			t.Errorf("UiDataTypeMap[%q] = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestUiDbTypeMap(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"text", "string"},
		{"integer", "unsignedBigInt"},
		{"boolean", "boolean"},
	}
	for _, tt := range tests {
		got, ok := UiDbTypeMap[tt.key]
		if !ok {
			t.Errorf("missing key %q in UiDbTypeMap", tt.key)
			continue
		}
		if got != tt.want {
			t.Errorf("UiDbTypeMap[%q] = %q, want %q", tt.key, got, tt.want)
		}
	}
}
