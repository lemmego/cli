package cli

import "testing"

func TestSnakeCaseValid(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{"user", true},
		{"user_profile", true},
		{"user123", true},
		{"my_resource_name", true},
		{"a", true},
		{"z_1", true},
		{"", false},
		{"User", false},
		{"user Profile", false},
		{"user-profile", false},
		{"_user", false},
		{"123abc", false},
	}
	for _, c := range cases {
		err := SnakeCase(c.input)
		if c.valid && err != nil {
			t.Errorf("expected %q to be valid, got %v", c.input, err)
		}
		if !c.valid && err == nil {
			t.Errorf("expected %q to be invalid", c.input)
		}
	}
}

func TestSnakeCaseEmptyAllowed(t *testing.T) {
	err := SnakeCaseEmptyAllowed("")
	if err != nil {
		t.Errorf("expected empty string to be allowed, got %v", err)
	}
	err = SnakeCaseEmptyAllowed("valid_name")
	if err != nil {
		t.Errorf("expected valid_name to be valid, got %v", err)
	}
	err = SnakeCaseEmptyAllowed("Invalid")
	if err == nil {
		t.Error("expected Invalid to fail")
	}
}

func TestNotIn(t *testing.T) {
	v := NotIn([]string{"reserved", "admin"}, "not allowed", SnakeCase)
	err := v("reserved")
	if err == nil {
		t.Error("expected error for reserved word")
	}
	err = v("admin")
	if err == nil {
		t.Error("expected error for admin")
	}
	err = v("user")
	if err != nil {
		t.Errorf("expected user to be allowed, got %v", err)
	}
}

func TestIn(t *testing.T) {
	v := In([]string{"a", "b", "c"}, "must be a, b, or c")
	err := v("x")
	if err != nil {
		t.Errorf("expected x to pass In (not in list), got %v", err)
	}
	err = v("a")
	if err == nil {
		t.Error("expected a to fail In (is in list)")
	}
}
