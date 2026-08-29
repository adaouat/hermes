package main

import "testing"

func TestParseProduct(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "known product", raw: "phpStorm", wantErr: false},
		{name: "unknown product", raw: "notAProduct", wantErr: true},
		{name: "empty string", raw: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseProduct(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseProduct(%q) error = %v, wantErr %v", tc.raw, err, tc.wantErr)
			}
			if err == nil && string(got) != tc.raw {
				t.Errorf("parseProduct(%q) = %q, want %q", tc.raw, got, tc.raw)
			}
		})
	}
}
