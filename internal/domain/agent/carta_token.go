package agent

import "strings"

const (
	TokenCarta     TokenType = "CARTA"
	TokenHuman     TokenType = "HUMAN"
	TokenGrounds   TokenType = "GROUNDS"
	TokenPermit    TokenType = "PERMIT"
	TokenDelegate  TokenType = "DELEGATE"
	TokenInvariant TokenType = "INVARIANT"
	TokenBudget    TokenType = "BUDGET"
	TokenSkill     TokenType = "SKILL"
	TokenNever     TokenType = "NEVER"
	TokenAlways    TokenType = "ALWAYS"
	TokenSlash     TokenType = "/"
)

var cartaKeywords = map[string]TokenType{
	"CARTA":     TokenCarta,
	"AGENT":     TokenAgent,
	"GROUNDS":   TokenGrounds,
	"PERMIT":    TokenPermit,
	"DELEGATE":  TokenDelegate,
	"INVARIANT": TokenInvariant,
	"BUDGET":    TokenBudget,
	"SKILL":     TokenSkill,
	"HUMAN":     TokenHuman,
	"NEVER":     TokenNever,
	"ALWAYS":    TokenAlways,
	"TO":        TokenTo,
	"WITH":      TokenWith,
	"IN":        TokenIn,
	"TRUE":      TokenBoolean,
	"FALSE":     TokenBoolean,
	"NULL":      TokenNull,
}

func IsCartaKeyword(s string) bool {
	_, ok := cartaKeywords[strings.TrimSpace(strings.ToUpper(s))]
	return ok
}
