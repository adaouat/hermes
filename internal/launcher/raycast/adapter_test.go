package raycast

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
)

func TestAdapter_Name(t *testing.T) {
	a := NewAdapter()
	if got := a.Name(); got != "raycast" {
		t.Errorf("Name() = %q, want %q", got, "raycast")
	}
}

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want bool
	}{
		{name: "RAYCAST_VERSION set", vars: map[string]string{"RAYCAST_VERSION": "1.80.0"}, want: true},
		{name: "RAYCAST_VERSION unset", vars: map[string]string{}, want: false},
		{name: "unrelated RAYCAST_ var doesn't count", vars: map[string]string{"RAYCAST_OTHER": "1"}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAdapter()
			if got := a.Detect(envtest.New(tc.vars)); got != tc.want {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAdapter_Render(t *testing.T) {
	a := NewAdapter()
	var buf bytes.Buffer
	if err := a.Render(nil, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if got.Items == nil || len(got.Items) != 0 {
		t.Errorf("Items = %+v, want empty non-nil slice", got.Items)
	}
}
