package agent

import (
	"strings"
	"unicode"
)

type CartaLexer struct{}

func NewCartaLexer() *CartaLexer {
	return &CartaLexer{}
}

func (l *CartaLexer) Lex(source string) ([]Token, error) {
	lines := normalizeLexerLines(source)
	tokens := make([]Token, 0, len(lines)*4)
	indentStack := []int{0}

	for lineIndex, rawLine := range lines {
		lineTokens, err := processCartaLexerLine(lineIndex, rawLine, &indentStack)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, lineTokens...)
	}

	tokens = append(tokens, emitFinalDedents(&indentStack, len(lines))...)
	tokens = append(tokens, Token{Type: TokenEOF, Literal: "", Line: maxLexerLine(len(lines)), Column: 1})
	return tokens, nil
}

func processCartaLexerLine(lineIndex int, rawLine string, indentStack *[]int) ([]Token, error) {
	lineNo := lineIndex + 1
	indentWidth, content := splitIndentation(rawLine)
	trimmed := strings.TrimSpace(content)

	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil, nil
	}

	indentTokens, err := emitIndentationTokens(lineNo, indentWidth, indentStack)
	if err != nil {
		return nil, err
	}

	lineTokens, err := lexCartaLine(content, lineNo, indentWidth+1)
	if err != nil {
		return nil, err
	}

	indentTokens = append(indentTokens, lineTokens...)
	indentTokens = append(indentTokens, Token{Type: TokenNewline, Literal: "\n", Line: lineNo, Column: len(rawLine) + 1})
	return indentTokens, nil
}

func lexCartaLine(line string, lineNo, startColumn int) ([]Token, error) {
	tokens := make([]Token, 0, len(line))
	for i := 0; i < len(line); {
		ch := rune(line[i])
		if unicode.IsSpace(ch) {
			i++
			continue
		}
		tok, next, err := lexNextCartaToken(line, i, lineNo, startColumn+i)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		i = next
	}
	return tokens, nil
}

func lexNextCartaToken(line string, i, lineNo, column int) (Token, int, error) {
	ch := rune(line[i])
	switch ch {
	case ':', ',', '[', ']', '{', '}':
		return lexSingleCharToken(ch, lineNo, column), i + 1, nil
	case '/':
		return Token{Type: TokenSlash, Literal: "/", Line: lineNo, Column: column}, i + 1, nil
	case '=', '!', '>', '<':
		tok, next, err := lexOperatorToken(line, ch, i, lineNo, column)
		return tok, next, err
	case '"':
		literal, next, err := readStringLiteral(line, i, lineNo, column)
		if err != nil {
			return Token{}, 0, err
		}
		return Token{Type: TokenString, Literal: literal, Line: lineNo, Column: column}, next, nil
	default:
		return lexCartaWordOrNumberToken(line, ch, i, lineNo, column)
	}
}

func lexCartaWordOrNumberToken(line string, ch rune, i, lineNo, column int) (Token, int, error) {
	if isIdentifierStart(ch) {
		literal, next := readIdentifier(line, i)
		return Token{Type: lookupCartaTokenType(literal), Literal: literal, Line: lineNo, Column: column}, next, nil
	}
	if isNumberStart(line, i) {
		literal, next := readNumber(line, i)
		return Token{Type: TokenNumber, Literal: literal, Line: lineNo, Column: column}, next, nil
	}
	return Token{}, 0, &LexerError{
		Line:   lineNo,
		Column: column,
		Reason: "unexpected character",
		Found:  Token{Type: TokenIllegal, Literal: string(ch), Line: lineNo, Column: column},
	}
}

func lookupCartaTokenType(literal string) TokenType {
	if tokenType, ok := cartaKeywords[strings.TrimSpace(strings.ToUpper(literal))]; ok {
		return tokenType
	}
	return TokenIdentifier
}
