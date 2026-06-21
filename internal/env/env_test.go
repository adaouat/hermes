package env

import (
	"reflect"
	"testing"
)

func TestNew_Lookup(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		setEnv    bool
		wantValue string
		wantOK    bool
	}{
		{name: "set var returns value and true", key: "HERMES_TEST_LOOKUP", value: "x", setEnv: true, wantValue: "x", wantOK: true},
		{name: "unset var returns empty and false", key: "HERMES_TEST_LOOKUP_UNSET", setEnv: false, wantValue: "", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setEnv {
				t.Setenv(tc.key, tc.value)
			}
			e := New()
			gotValue, gotOK := e.Lookup(tc.key)
			if gotValue != tc.wantValue || gotOK != tc.wantOK {
				t.Errorf("Lookup(%q) = (%q, %v), want (%q, %v)", tc.key, gotValue, gotOK, tc.wantValue, tc.wantOK)
			}
		})
	}
}

func TestNew_Home(t *testing.T) {
	t.Setenv("HOME", "/Users/x")
	e := New()
	if got := e.Home(); got != "/Users/x" {
		t.Errorf("Home() = %q, want %q", got, "/Users/x")
	}
}

func TestNew_Home_unset(t *testing.T) {
	t.Setenv("HOME", "")
	e := New()
	if got := e.Home(); got != "" {
		t.Errorf("Home() = %q, want empty string", got)
	}
}

func TestNew_Path(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "multiple entries", value: "/usr/bin:/bin:/opt/homebrew/bin", want: []string{"/usr/bin", "/bin", "/opt/homebrew/bin"}},
		{name: "single entry", value: "/usr/bin", want: []string{"/usr/bin"}},
		{name: "empty value returns nil", value: "", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", tc.value)
			e := New()
			got := e.Path()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Path() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
