package agent

import "strings"

type TokenType string

const (
	TokenIllegal TokenType = "ILLEGAL"
	TokenEOF     TokenType = "EOF"
	TokenNewline TokenType = "NEWLINE"
	TokenIndent  TokenType = "INDENT"
	TokenDedent  TokenType = "DEDENT"

	TokenIdentifier TokenType = "IDENT"
	TokenString     TokenType = "STRING"
	TokenNumber     TokenType = "NUMBER"
	TokenBoolean    TokenType = "BOOLEAN"
	TokenNull       TokenType = "NULL"

	TokenAssign   TokenType = "="
	TokenEqual    TokenType = "=="
	TokenNotEqual TokenType = "!="
	TokenGT       TokenType = ">"
	TokenLT       TokenType = "<"
	TokenGTE      TokenType = ">="
	TokenLTE      TokenType = "<="
	TokenColon    TokenType = ":"
	TokenComma    TokenType = ","
	TokenLBracket TokenType = "["
	TokenRBracket TokenType = "]"
	TokenLBrace   TokenType = "{"
	TokenRBrace   TokenType = "}"

	TokenWorkflow TokenType = "WORKFLOW"
	TokenOn       TokenType = "ON"
	TokenIf       TokenType = "IF"
	TokenSet      TokenType = "SET"
	TokenNotify   TokenType = "NOTIFY"
	TokenWith     TokenType = "WITH"
	TokenTo       TokenType = "TO"
	TokenAgent    TokenType = "AGENT"
	TokenIn       TokenType = "IN"

	TokenWait     TokenType = "WAIT"
	TokenDispatch TokenType = "DISPATCH"
	TokenSurface  TokenType = "SURFACE"
	TokenAs       TokenType = "AS"   // CLSF-52
	TokenRole     TokenType = "ROLE" // CLSF-53

	// DSL v1 reserved keywords — lexed but not yet executable. // CLSF-50
	TokenCall    TokenType = "CALL"
	TokenApprove TokenType = "APPROVE"
)

type Token struct {
	Type    TokenType `json:"type"`
	Literal string    `json:"literal"`
	Line    int       `json:"line"`
	Column  int       `json:"column"`
}

var dslKeywords = map[string]TokenType{
	"WORKFLOW": TokenWorkflow,
	"ON":       TokenOn,
	"IF":       TokenIf,
	"SET":      TokenSet,
	"NOTIFY":   TokenNotify,
	"WITH":     TokenWith,
	"TO":       TokenTo,
	"AGENT":    TokenAgent,
	"IN":       TokenIn,
	"WAIT":     TokenWait,
	"DISPATCH": TokenDispatch,
	"SURFACE":  TokenSurface,
	"AS":       TokenAs,   // CLSF-52
	"ROLE":     TokenRole, // CLSF-53
	"TRUE":     TokenBoolean,
	"FALSE":    TokenBoolean,
	"NULL":     TokenNull,
}

// dslReservedKeywords are lexed but not yet executable in DSL v0. // CLSF-50
var dslReservedKeywords = map[string]TokenType{
	"CALL":    TokenCall,
	"APPROVE": TokenApprove,
}

func LookupTokenType(literal string) TokenType {
	normalized := strings.TrimSpace(strings.ToUpper(literal))
	if tokenType, ok := dslKeywords[normalized]; ok {
		return tokenType
	}
	if tokenType, ok := dslReservedKeywords[normalized]; ok {
		return tokenType
	}
	return TokenIdentifier
}

func IsKeyword(tokenType TokenType) bool {
	for _, candidate := range dslKeywords {
		if candidate == tokenType {
			return true
		}
	}
	return false
}

func IsReservedKeyword(tokenType TokenType) bool {
	for _, candidate := range dslReservedKeywords {
		if candidate == tokenType {
			return true
		}
	}
	return false
}

func IsLiteralToken(tokenType TokenType) bool {
	switch tokenType {
	case TokenIdentifier, TokenString, TokenNumber, TokenBoolean, TokenNull:
		return true
	default:
		return false
	}
}

func IsComparisonOperator(tokenType TokenType) bool {
	switch tokenType {
	case TokenEqual, TokenNotEqual, TokenGT, TokenLT, TokenGTE, TokenLTE, TokenIn:
		return true
	default:
		return false
	}
}

func IsStructuralToken(tokenType TokenType) bool {
	switch tokenType {
	case TokenNewline, TokenIndent, TokenDedent, TokenColon, TokenComma, TokenLBracket, TokenRBracket, TokenLBrace, TokenRBrace:
		return true
	default:
		return false
	}
}
