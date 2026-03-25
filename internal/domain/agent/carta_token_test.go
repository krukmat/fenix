package agent

import (
	"strings"
	"testing"
)

func TestCartaTokenRecognizesKeywords(t *testing.T) {
	t.Parallel()

	cases := map[string]TokenType{
		"CARTA":     TokenCarta,
		"agent":     TokenAgent,
		"Grounds":   TokenGrounds,
		"PERMIT":    TokenPermit,
		"delegate":  TokenDelegate,
		"invariant": TokenInvariant,
		"BUDGET":    TokenBudget,
		"skill":     TokenSkill,
		"human":     TokenHuman,
		"never":     TokenNever,
		"always":    TokenAlways,
		"to":        TokenTo,
		"WITH":      TokenWith,
	}

	for input, want := range cases {
		got, ok := cartaKeywords[strings.ToUpper(input)]
		if !ok {
			t.Fatalf("cartaKeywords missing %q", input)
		}
		if got != want {
			t.Fatalf("cartaKeywords[%q] = %s, want %s", input, got, want)
		}
	}
}

func TestCartaTokenIsCartaKeyword(t *testing.T) {
	t.Parallel()

	if !IsCartaKeyword("GROUNDS") {
		t.Fatal("expected GROUNDS to be a Carta keyword")
	}
	if IsCartaKeyword("IF") {
		t.Fatal("expected IF not to be a Carta keyword")
	}
	if IsCartaKeyword("resolve_support_case") {
		t.Fatal("expected regular identifier not to be a Carta keyword")
	}
}
