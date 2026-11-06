package contracts

import "testing"

func TestKeywordID(t *testing.T) {
	cases := []struct{ group, lang, spelling, want string }{
		{"baghdad", "ar", "بغداد", "baghdad:ar:بغداد"},
		// Diacritics and tatweel go; alef with hamza becomes bare alef.
		{"baghdad", "ar", "بَغْدَاد", "baghdad:ar:بغداد"},
		{"basra", "ar", "البصـــرة", "basra:ar:البصره"},
		{"iraq", "ar", "أَلعِراق", "iraq:ar:العراق"},
		// Alef maksura and Farsi yeh are the same yeh; keheh is kaf.
		{"ali", "ar", "علی", "ali:ar:علي"},
		{"karbala", "ar", "کربلاء", "karbala:ar:كربلاء"},
		// Kurdish letters stay.
		{"hawler", "ckb", "هەولێر", "hawler:ckb:هەولێر"},
		// Latin: lower case, spaces collapsed; group and language lower-cased.
		{"Erbil", "EN", "  Erbil   Citadel ", "erbil:en:erbil citadel"},
		// Arabic presentation forms (U+FExx) fold to ordinary letters.
		{"iraq", "ar", "ﺍﻟﻌﺮﺍﻖ", "iraq:ar:العراق"},
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
		"baghdad:ar: بغداد", "baghdad:ar:بَغْدَاد", "erbil:en:Erbil",
	} {
		if ValidKeywordID(id) {
			t.Errorf("ValidKeywordID(%q) = true", id)
		}
	}
}
