package domain

import (
	"slices"
	"testing"
)

func TestProducts_count(t *testing.T) {
	if got := len(Products()); got != 19 {
		t.Errorf("len(Products()) = %d, want 19", got)
	}
}

func TestProducts_containsExpected(t *testing.T) {
	want := []Product{AndroidStudio, IntelliJIdeaUltimate, PhpStorm, WebStorm, Fleet}
	products := Products()
	for _, w := range want {
		if !slices.Contains(products, w) {
			t.Errorf("Products() does not contain %q", w)
		}
	}
}

func TestProduct_Display(t *testing.T) {
	tests := []struct {
		name    string
		product Product
		want    string
	}{
		{name: "phpStorm", product: PhpStorm, want: "PhpStorm"},
		{name: "webStorm", product: WebStorm, want: "WebStorm"},
		{name: "androidStudio", product: AndroidStudio, want: "AndroidStudio"},
		{name: "cLion", product: CLion, want: "CLion"},
		{name: "cLionNova", product: CLionNova, want: "CLionNova"},
		{name: "intelliJIdeaCommunity", product: IntelliJIdeaCommunity, want: "IntelliJIdeaCommunity"},
		{name: "intelliJIdeaUltimate", product: IntelliJIdeaUltimate, want: "IntelliJIdeaUltimate"},
		{name: "pyCharmProfessional", product: PyCharmProfessional, want: "PyCharmProfessional"},
		{name: "pyCharmCommunity", product: PyCharmCommunity, want: "PyCharmCommunity"},
		{name: "fleet", product: Fleet, want: "Fleet"},
		{name: "rubyMine", product: RubyMine, want: "RubyMine"},
		{name: "rustRover", product: RustRover, want: "RustRover"},
		{name: "goLand", product: GoLand, want: "GoLand"},
		{name: "dataGrip", product: DataGrip, want: "DataGrip"},
		{name: "dataSpell", product: DataSpell, want: "DataSpell"},
		{name: "appCode", product: AppCode, want: "AppCode"},
		{name: "aqua", product: Aqua, want: "Aqua"},
		{name: "rider", product: Rider, want: "Rider"},
		{name: "writerside", product: Writerside, want: "Writerside"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.product.Display(); got != tc.want {
				t.Errorf("Display() = %q, want %q", got, tc.want)
			}
		})
	}
}

// Regression test for the legacy toJbName bug ([bug] toJbName mishandles
// already-capitalised input): 'PhpStorm'.toJbName() incorrectly returned
// 'PHpStorm' because the Dart implementation only stopped capitalising after it
// actually converted a lowercase letter. Display() must capitalise only the
// first rune and leave the rest untouched, regardless of its starting case.
func TestProduct_Display_doesNotDoubleCapitalize(t *testing.T) {
	tests := []struct {
		name    string
		product Product
		want    string
	}{
		{name: "already capitalised", product: Product("PhpStorm"), want: "PhpStorm"},
		{name: "all caps", product: Product("PHPSTORM"), want: "PHPSTORM"},
		{name: "empty string", product: Product(""), want: ""},
		{name: "single lowercase char", product: Product("a"), want: "A"},
		{name: "single uppercase char", product: Product("A"), want: "A"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.product.Display(); got != tc.want {
				t.Errorf("Display() = %q, want %q", got, tc.want)
			}
		})
	}
}
