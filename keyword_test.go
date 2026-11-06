package contracts

import "testing"

func TestKeywordID(t *testing.T) {
	cases := []struct{ group, lang, spelling, want string }{
		{"baghdad", "ar", "بغداد", "baghdad:ar:بغداد"},
		// Diacritics and tatweel go; alef with hamza becomes bare alef.
		{"baghdad", "ar", "بَغْدَاد", "baghdad:ar:بغداد"},
		{"basra", "ar", "البصـــرة", "basra:ar:البصره"},
		{"iraq", "ar", "أَلعِراق", "iraq:ar:العراق"},
		// Arabic yeh and alef maksura become Kurdish yeh; kaf becomes keheh.
		{"ali", "ar", "علي", "ali:ar:علی"},
		{"karbala", "ar", "كربلاء", "karbala:ar:کربلاء"},
		// Sorani typed with ە, or with ه and a ZWNJ, gives one ID.
		{"hawler", "ckb", "هەولێر", "hawler:ckb:ههولێر"},
		{"hawler", "ckb", "هه‌ولێر", "hawler:ckb:ههولێر"},
		// Latin: lower case, accents, apostrophes, hyphens, spaces.
		{"Erbil", "EN", "  Erbil   Citadel ", "erbil:en:erbil citadel"},
		{"baquba", "en", "Ba'quba", "baquba:en:baquba"},
		{"qaim", "en", "Al-Qaim", "qaim:en:al qaim"},
		{"duhok", "kmr", "Dihêk", "duhok:kmr:dihek"},
		// Arabic-Indic digits become ASCII.
		{"phone", "ar", "٩٦٤", "phone:ar:964"},
	}
	for _, c := range cases {
		if got := KeywordID(c.group, c.lang, c.spelling); got != c.want {
			t.Errorf("KeywordID(%q, %q, %q) = %q, want %q", c.group, c.lang, c.spelling, got, c.want)
		}
		if !ValidKeywordID(c.want) {
			t.Errorf("ValidKeywordID(%q) = false", c.want)
		}
	}
}

func TestValidKeywordID(t *testing.T) {
	for _, id := range []string{
		"", "baghdad", "baghdad:ar", "Baghdad:ar:بغداد", "baghdad:arabic:بغداد",
		"baghdad:ar: بغداد", "baghdad:ar:بَغْدَاد", "erbil:en:Erbil", "ali:ar:علي",
	} {
		if ValidKeywordID(id) {
			t.Errorf("ValidKeywordID(%q) = true", id)
		}
	}
}
