package contracts

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// KeywordID is a keyword's ID in the shared keyword list:
// "<group>:<language>:<folded spelling>", for example
// "baghdad:ar:بغداد". Every service computes it the same way, so the same
// word has the same ID everywhere, with no central counter. Discovery's
// Evidence.keyword_id and Relevance.matched_keyword_ids use it.
//
// group and language are lower-case ASCII ("baghdad", "ar", "ckb",
// "kmr", "en"). The spelling is folded with FoldKeyword.
func KeywordID(group, language, spelling string) string {
	return asciiLower(group) + ":" + asciiLower(language) + ":" + FoldKeyword(spelling)
}

// FoldKeyword folds a spelling so that variants a person would read as the
// same word match:
//   - Unicode NFKC (Arabic presentation forms become ordinary letters);
//   - lower case;
//   - Arabic diacritics (harakat, shadda, sukun, superscript alef) and
//     tatweel removed;
//   - alef with hamza or madda, and alef wasla, become a bare alef (ا);
//   - teh marbuta (ة) becomes heh (ه), alef maksura (ى) and Farsi yeh (ی)
//     become yeh (ي), keheh (ک) becomes kaf (ك);
//   - Kurdish letters (ڕ ڵ ۆ ێ ە) are kept as they are;
//   - spaces collapsed to one, and trimmed.
func FoldKeyword(s string) string {
	s = norm.NFKC.String(s)
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			space = b.Len() > 0
			continue
		}
		if isArabicMark(r) {
			continue
		}
		if f, ok := arabicFold[r]; ok {
			r = f
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

var arabicFold = map[rune]rune{
	'أ': 'ا', 'إ': 'ا', 'آ': 'ا', 'ٱ': 'ا',
	'ة': 'ه',
	'ى': 'ي', 'ی': 'ي',
	'ک': 'ك',
}

// isArabicMark: harakat and other combining marks (U+064B–U+065F),
// superscript alef (U+0670), Quranic marks (U+06D6–U+06ED) and tatweel.
func isArabicMark(r rune) bool {
	return (r >= 0x064B && r <= 0x065F) || r == 0x0670 || (r >= 0x06D6 && r <= 0x06ED) || r == 0x0640
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
