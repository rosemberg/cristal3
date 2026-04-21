package htmlextract

import (
	"sort"
	"strings"
	"unicode"
)

// ptBRStopwords is the set of common Portuguese stopwords to filter out.
var ptBRStopwords = map[string]struct{}{
	"a": {}, "o": {}, "as": {}, "os": {}, "e": {}, "de": {}, "do": {}, "da": {},
	"dos": {}, "das": {}, "em": {}, "no": {}, "na": {}, "nos": {}, "nas": {},
	"para": {}, "por": {}, "com": {}, "sem": {}, "que": {}, "se": {}, "ou": {},
	"um": {}, "uma": {}, "mais": {}, "menos": {}, "isto": {}, "isso": {},
	"ele": {}, "ela": {}, "eles": {}, "elas": {}, "você": {}, "nós": {},
	"seu": {}, "sua": {}, "seus": {}, "suas": {}, "muito": {}, "tudo": {},
	"todo": {}, "toda": {}, "também": {}, "então": {}, "quando": {}, "onde": {},
	"como": {}, "qual": {}, "quais": {}, "até": {}, "sobre": {}, "entre": {},
	"após": {}, "antes": {}, "depois": {}, "ser": {}, "ter": {}, "estar": {},
	"foi": {}, "são": {}, "está": {}, "estão": {}, "pela": {}, "pelo": {},
	"pelas": {}, "pelos": {}, "ao": {}, "aos": {}, "à": {}, "às": {},
	"este": {}, "esta": {}, "estes": {}, "estas": {}, "esse": {}, "essa": {},
	"esses": {}, "essas": {}, "aquele": {}, "aquela": {}, "aqueles": {}, "aquelas": {},
	"num": {}, "numa": {}, "nuns": {}, "numas": {}, "deste": {}, "desta": {},
	"neste": {}, "nesta": {}, "nesses": {}, "nessas": {}, "desse": {}, "dessa": {},
	"desses": {}, "dessas": {}, "daquele": {}, "daquela": {}, "naquele": {}, "naquela": {},
	"des": {}, "from": {}, "the": {}, "and": {}, "for": {}, "with": {},
}

// extractKeywords tokenizes fullText, removes stopwords, and returns
// the top 10 tokens by frequency (ties broken alphabetically).
func extractKeywords(fullText string) []string {
	if len(fullText) < 50 {
		return nil
	}

	// Tokenize: lowercase, split on non-letter-or-digit runes, keep length >= 3
	freq := make(map[string]int)
	var token strings.Builder
	flush := func() {
		if token.Len() >= 3 {
			w := token.String()
			if _, stop := ptBRStopwords[w]; !stop {
				freq[w]++
			}
		}
		token.Reset()
	}

	for _, r := range strings.ToLower(fullText) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			token.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()

	if len(freq) == 0 {
		return nil
	}

	// Sort by count desc, then alphabetically
	type kv struct {
		key   string
		count int
	}
	kvs := make([]kv, 0, len(freq))
	for k, v := range freq {
		kvs = append(kvs, kv{k, v})
	}
	sort.Slice(kvs, func(i, j int) bool {
		if kvs[i].count != kvs[j].count {
			return kvs[i].count > kvs[j].count
		}
		return kvs[i].key < kvs[j].key
	})

	// Top 10
	limit := 10
	if len(kvs) < limit {
		limit = len(kvs)
	}
	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = kvs[i].key
	}
	return result
}
