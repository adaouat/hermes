package envtest

import (
	"reflect"
	"testing"
)

func TestNew_Lookup(t *testing.T) {
	tests := []struct {
		name      string
		vars      map[string]string
		key       string
		wantValue string
		wantOK    bool
	}{
		{name: "present key", vars: map[string]string{"HOME": "/Users/x"}, key: "HOME", wantValue: "/Users/x", wantOK: true},
		{name: "absent key", vars: map[string]string{"HOME": "/Users/x"}, key: "PATH", wantValue: "", wantOK: false},
		{name: "nil map", vars: nil, key: "HOME", wantValue: "", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := New(tc.vars)
			gotValue, gotOK := e.Lookup(tc.key)
			if gotValue != tc.wantValue || gotOK != tc.wantOK {
				t.Errorf("Lookup(%q) = (%q, %v), want (%q, %v)", tc.key, gotValue, gotOK, tc.wantValue, tc.wantOK)
			}
		})
	}
}

func TestNew_Home(t *testing.T) {
	e := New(map[string]string{"HOME": "/Users/x"})
	if got := e.Home(); got != "/Users/x" {
		t.Errorf("Home() = %q, want %q", got, "/Users/x")
	}
}

func TestNew_Path(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want []string
	}{
		{name: "multiple entries", vars: map[string]string{"PATH": "/usr/bin:/bin"}, want: []string{"/usr/bin", "/bin"}},
		{name: "unset PATH", vars: map[string]string{}, want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := New(tc.vars)
			got := e.Path()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Path() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
