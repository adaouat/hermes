package domain

import "unicode"

// Product identifies a supported JetBrains IDE. Values match the legacy CLI's
// --product flag values (lowerCamelCase) for backwards compatibility.
type Product string

const (
	AndroidStudio         Product = "androidStudio"
	AppCode               Product = "appCode"
	Aqua                  Product = "aqua"
	CLion                 Product = "cLion"
	CLionNova             Product = "cLionNova"
	DataGrip              Product = "dataGrip"
	DataSpell             Product = "dataSpell"
	Fleet                 Product = "fleet"
	GoLand                Product = "goLand"
	IntelliJIdeaCommunity Product = "intelliJIdeaCommunity"
	IntelliJIdeaUltimate  Product = "intelliJIdeaUltimate"
	PhpStorm              Product = "phpStorm"
	PyCharmProfessional   Product = "pyCharmProfessional"
	PyCharmCommunity      Product = "pyCharmCommunity"
	Rider                 Product = "rider"
	RubyMine              Product = "rubyMine"
	RustRover             Product = "rustRover"
	WebStorm              Product = "webStorm"
	Writerside            Product = "writerside"
)

// Products returns every supported Product, in the same order as the legacy CLI's enum.
func Products() []Product {
	return []Product{
		AndroidStudio,
		AppCode,
		Aqua,
		CLion,
		CLionNova,
		DataGrip,
		DataSpell,
		Fleet,
		GoLand,
		IntelliJIdeaCommunity,
		IntelliJIdeaUltimate,
		PhpStorm,
		PyCharmProfessional,
		PyCharmCommunity,
		Rider,
		RubyMine,
		RustRover,
		WebStorm,
		Writerside,
	}
}

// Display returns the human-readable name for p, capitalizing only the first rune.
//
// The legacy Dart CLI's toJbName() walked the whole string and re-capitalized after
// every already-uppercase letter, so calling it on already-capitalized input doubled up
// (e.g. "PhpStorm" -> "PHpStorm"). Display never re-examines characters after the first,
// so it can't reproduce that class of bug.
func (p Product) Display() string {
	s := string(p)
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
