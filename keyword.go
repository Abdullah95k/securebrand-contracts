package contracts

import (
	"regexp"
	"strings"
	"unicode"
)

// KeywordID is a keyword's ID in the shared keyword list:
// "<group>:<language>:<folded spelling>", for example "baghdad:ar:بغداد".
// Every service computes it the same way, so the same keyword has the same
// ID everywhere, with no central counter. Discovery's Evidence.keyword_id
// and Relevance.matched_keyword_ids use it.
//
// Compute it from the keyword list's stored spelling, never from the text
// that matched: two spellings a fold doesn't merge would give two IDs.
// group and language are lower-case ASCII ("baghdad"; "en", "ar", "ckb",
// "kmr"). The spelling is folded with FoldKeyword.
func KeywordID(group, language, spelling string) string {
	return asciiLower(group) + ":" + asciiLower(language) + ":" + FoldKeyword(spelling)
}

// FoldKeyword is the spelling fold fbs and igs already share
// (fbs internal/discovery, igs internal/keywords Normalize), so IDs agree
// with what they match. Services may call this instead of their own copy.
//
//   - Arabic script: harakat, the superscript alef, tatweel and zero-width
//     joiners and marks are dropped (Sorani is often typed with a ZWNJ inside
//     a word: هه‌ولێر); أ إ آ ٱ fold to ا; ة and Sorani ە fold to ه; ي ى and
//     ی fold to ی, and ك to ک, the forms Kurdish uses, so Arabic and Kurdish
//     keyboards agree. Kurdish letters of their own (ێ ۆ ڵ ڕ ڤ گ چ پ ژ) are
//     kept. Arabic-Indic digits become ASCII.
//   - Latin script: lower case; accents folded (ê→e, î→i, û→u, ç→c, ş→s,
//     ğ→g, ı→i, and the common Western accents); apostrophes dropped
//     (Ba'quba, Baquba); hyphens and underscores become spaces.
//   - Whitespace collapsed and trimmed.
//
// The Arabic article ال is not stripped.
func FoldKeyword(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		r = foldRune(r)
		switch {
		case r == dropRune:
			continue
		case unicode.IsSpace(r):
			space = b.Len() > 0
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

const dropRune = rune(-1)

func foldRune(r rune) rune {
	switch {
	case r >= 0x064B && r <= 0x065F, r == 0x0670, r == 0x0640, // harakat, superscript alef, tatweel
		r == 0x200C, r == 0x200D, r == 0x200E, r == 0x200F, r == 0xFEFF: // ZWNJ, ZWJ, marks, BOM
		return dropRune
	case r == '\'', r == '’', r == 'ʼ', r == '‘', r == '`':
		return dropRune
	case r == '-', r == '_', r == 0x2010, r == 0x2011, r == 0x2013:
		return ' '
	case r >= 0x0660 && r <= 0x0669: // Arabic-Indic digits
		return '0' + (r - 0x0660)
	case r >= 0x06F0 && r <= 0x06F9: // Extended Arabic-Indic digits
		return '0' + (r - 0x06F0)
	}
	switch r {
	case 'أ', 'إ', 'آ', 'ٱ':
		return 'ا'
	case 'ة', 'ە':
		return 'ه'
	case 'ي', 'ى':
		return 'ی'
	case 'ك':
		return 'ک'
	}
	if f, ok := latinFold[r]; ok {
		return f
	}
	return unicode.ToLower(r)
}

var latinFold = map[rune]rune{
	'ê': 'e', 'Ê': 'e', 'é': 'e', 'É': 'e', 'è': 'e', 'È': 'e', 'ë': 'e', 'Ë': 'e',
	'î': 'i', 'Î': 'i', 'í': 'i', 'Í': 'i', 'ì': 'i', 'ï': 'i', 'ı': 'i', 'İ': 'i',
	'û': 'u', 'Û': 'u', 'ú': 'u', 'Ú': 'u', 'ù': 'u', 'ü': 'u', 'Ü': 'u',
	'â': 'a', 'Â': 'a', 'á': 'a', 'Á': 'a', 'à': 'a', 'ä': 'a', 'Ä': 'a',
	'ô': 'o', 'Ô': 'o', 'ó': 'o', 'Ó': 'o', 'ò': 'o', 'ö': 'o', 'Ö': 'o',
	'ç': 'c', 'Ç': 'c', 'ş': 's', 'Ş': 's', 'ğ': 'g', 'Ğ': 'g',
}

func asciiLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

var keywordIDRE = regexp.MustCompile(`^[a-z0-9_-]+:[a-z]{2,3}(-[a-z0-9]+)?:\S(.*\S)?$`)

// ValidKeywordID reports whether id has KeywordID's form, with a spelling
// that is already folded.
func ValidKeywordID(id string) bool {
	if !keywordIDRE.MatchString(id) {
		return false
	}
	parts := strings.SplitN(id, ":", 3)
	return FoldKeyword(parts[2]) == parts[2]
}
