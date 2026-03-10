package agent

import (
	"fmt"
	"strings"
	"unicode"
)

type Lexer struct {
	source string
}

func NewLexer(source string) *Lexer {
	return &Lexer{source: source}
}

func (l *Lexer) Lex() ([]Token, error) {
	lines := normalizeLexerLines(l.source)
	tokens := make([]Token, 0, len(lines)*4)
	indentStack := []int{0}

	for lineIndex, rawLine := range lines {
		lineNo := lineIndex + 1
		indentWidth, content := splitIndentation(rawLine)
		trimmed := strings.TrimSpace(content)

		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		nextTokens, err := emitIndentationTokens(lineNo, indentWidth, &indentStack)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, nextTokens...)

		lineTokens, err := lexLine(content, lineNo, indentWidth+1)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, lineTokens...)
		tokens = append(tokens, Token{
			Type:    TokenNewline,
			Literal: "\n",
			Line:    lineNo,
			Column:  len(rawLine) + 1,
		})
	}

	for len(indentStack) > 1 {
		indentStack = indentStack[:len(indentStack)-1]
		tokens = append(tokens, Token{
			Type:    TokenDedent,
			Literal: "",
			Line:    len(lines),
			Column:  1,
		})
	}

	tokens = append(tokens, Token{
		Type:    TokenEOF,
		Literal: "",
		Line:    maxLexerLine(len(lines)),
		Column:  1,
	})
	return tokens, nil
}

func normalizeLexerLines(source string) []string {
	normalized := strings.ReplaceAll(source, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

func splitIndentation(line string) (int, string) {
	width := 0
	for width < len(line) && line[width] == ' ' {
		width++
	}
	return width, line[width:]
}

func emitIndentationTokens(lineNo, indentWidth int, indentStack *[]int) ([]Token, error) {
	tokens := make([]Token, 0, 2)
	current := (*indentStack)[len(*indentStack)-1]

	switch {
	case indentWidth > current:
		*indentStack = append(*indentStack, indentWidth)
		tokens = append(tokens, Token{
			Type:    TokenIndent,
			Literal: "",
			Line:    lineNo,
			Column:  1,
		})
	case indentWidth < current:
		for len(*indentStack) > 1 && indentWidth < (*indentStack)[len(*indentStack)-1] {
			*indentStack = (*indentStack)[:len(*indentStack)-1]
			tokens = append(tokens, Token{
				Type:    TokenDedent,
				Literal: "",
				Line:    lineNo,
				Column:  1,
			})
		}
		if indentWidth != (*indentStack)[len(*indentStack)-1] {
			return nil, &LexerError{
				Line:   lineNo,
				Column: 1,
				Reason: "inconsistent indentation",
			}
		}
	}

	return tokens, nil
}

func lexLine(line string, lineNo, startColumn int) ([]Token, error) {
	tokens := make([]Token, 0, len(line))
	for i := 0; i < len(line); {
		ch := rune(line[i])
		column := startColumn + i

		if unicode.IsSpace(ch) {
			i++
			continue
		}

		switch ch {
		case ':':
			tokens = append(tokens, Token{Type: TokenColon, Literal: ":", Line: lineNo, Column: column})
			i++
		case ',':
			tokens = append(tokens, Token{Type: TokenComma, Literal: ",", Line: lineNo, Column: column})
			i++
		case '[':
			tokens = append(tokens, Token{Type: TokenLBracket, Literal: "[", Line: lineNo, Column: column})
			i++
		case ']':
			tokens = append(tokens, Token{Type: TokenRBracket, Literal: "]", Line: lineNo, Column: column})
			i++
		case '{':
			tokens = append(tokens, Token{Type: TokenLBrace, Literal: "{", Line: lineNo, Column: column})
			i++
		case '}':
			tokens = append(tokens, Token{Type: TokenRBrace, Literal: "}", Line: lineNo, Column: column})
			i++
		case '=':
			if i+1 < len(line) && line[i+1] == '=' {
				tokens = append(tokens, Token{Type: TokenEqual, Literal: "==", Line: lineNo, Column: column})
				i += 2
				continue
			}
			tokens = append(tokens, Token{Type: TokenAssign, Literal: "=", Line: lineNo, Column: column})
			i++
		case '!':
			if i+1 < len(line) && line[i+1] == '=' {
				tokens = append(tokens, Token{Type: TokenNotEqual, Literal: "!=", Line: lineNo, Column: column})
				i += 2
				continue
			}
			return nil, &LexerError{Line: lineNo, Column: column, Reason: "unexpected character '!'"}
		case '>':
			if i+1 < len(line) && line[i+1] == '=' {
				tokens = append(tokens, Token{Type: TokenGTE, Literal: ">=", Line: lineNo, Column: column})
				i += 2
				continue
			}
			tokens = append(tokens, Token{Type: TokenGT, Literal: ">", Line: lineNo, Column: column})
			i++
		case '<':
			if i+1 < len(line) && line[i+1] == '=' {
				tokens = append(tokens, Token{Type: TokenLTE, Literal: "<=", Line: lineNo, Column: column})
				i += 2
				continue
			}
			tokens = append(tokens, Token{Type: TokenLT, Literal: "<", Line: lineNo, Column: column})
			i++
		case '"':
			literal, next, err := readStringLiteral(line, i, lineNo, column)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{Type: TokenString, Literal: literal, Line: lineNo, Column: column})
			i = next
		default:
			switch {
			case isIdentifierStart(ch):
				literal, next := readIdentifier(line, i)
				tokens = append(tokens, Token{
					Type:    LookupTokenType(literal),
					Literal: literal,
					Line:    lineNo,
					Column:  column,
				})
				i = next
			case isNumberStart(line, i):
				literal, next := readNumber(line, i)
				tokens = append(tokens, Token{
					Type:    TokenNumber,
					Literal: literal,
					Line:    lineNo,
					Column:  column,
				})
				i = next
			default:
				return nil, &LexerError{
					Line:   lineNo,
					Column: column,
					Reason: fmt.Sprintf("unexpected character %q", ch),
					Found:  Token{Type: TokenIllegal, Literal: string(ch), Line: lineNo, Column: column},
				}
			}
		}
	}

	return tokens, nil
}

func readStringLiteral(line string, start, lineNo, column int) (string, int, error) {
	for i := start + 1; i < len(line); i++ {
		if line[i] == '"' && line[i-1] != '\\' {
			return line[start : i+1], i + 1, nil
		}
	}
	return "", 0, &LexerError{
		Line:   lineNo,
		Column: column,
		Reason: "unterminated string literal",
		Found:  Token{Type: TokenString, Literal: line[start:], Line: lineNo, Column: column},
	}
}

func readIdentifier(line string, start int) (string, int) {
	i := start
	for i < len(line) {
		ch := rune(line[i])
		if !isIdentifierPart(ch) {
			break
		}
		i++
	}
	return line[start:i], i
}

func readNumber(line string, start int) (string, int) {
	i := start
	dotSeen := false
	if line[i] == '-' {
		i++
	}
	for i < len(line) {
		ch := rune(line[i])
		switch {
		case unicode.IsDigit(ch):
			i++
		case ch == '.' && !dotSeen:
			dotSeen = true
			i++
		default:
			return line[start:i], i
		}
	}
	return line[start:i], i
}

func isIdentifierStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentifierPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '.'
}

func isNumberStart(line string, index int) bool {
	if index >= len(line) {
		return false
	}
	ch := rune(line[index])
	if unicode.IsDigit(ch) {
		return true
	}
	return ch == '-' && index+1 < len(line) && unicode.IsDigit(rune(line[index+1]))
}

func maxLexerLine(lineCount int) int {
	if lineCount <= 0 {
		return 1
	}
	return lineCount
}
