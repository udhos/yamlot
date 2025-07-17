// Package token finds tokens.
package token

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Tokenizer tokenizes yaml tokens.
type Tokenizer struct {
	reader *bufio.Reader
	line   int
	column int
	status tokenStatus
}

type tokenStatus int

const (
	statusBlank tokenStatus = iota
	statusAfterDash
)

var statusName = []string{
	"StatusBlank",
	"StatusAfterDash",
}

// NewTokenizer creates tokenizer.
func NewTokenizer(input io.Reader) *Tokenizer {
	return &Tokenizer{
		reader: bufio.NewReader(input),
		line:   1,
		column: 0,
		status: statusBlank,
	}
}

func (t *Tokenizer) returnEOF() (Token, error) {
	return Token{Type: TokenEOF, Line: t.line, Column: t.column}, io.EOF
}

func (t *Tokenizer) returnNewLine() (Token, error) {
	tk := Token{Type: TokenNewLine, Value: "\\n", Line: t.line, Column: t.column}
	t.line++
	t.column = 0
	return tk, nil
}

func (t *Tokenizer) returnDash() (Token, error) {
	tk := Token{Type: TokenDash, Value: "-", Line: t.line, Column: t.column}
	t.status = statusAfterDash
	return tk, nil
}

func (t *Tokenizer) unreadAndReturnDash() (Token, error) {
	if err := t.reader.UnreadRune(); err != nil {
		return Token{}, err
	}
	return t.returnDash()
}

func (t *Tokenizer) collectPlainScalar(scalar []rune) (Token, error) {

	for {
		peek, err := t.reader.Peek(1)
		if err == io.EOF {
			break
		}
		if err != nil {
			return Token{}, err
		}

		if peek[0] == '\n' || peek[0] == '#' {
			break
		}
		ch, _, err := t.reader.ReadRune()
		if err != nil {
			return Token{}, err
		}
		scalar = append(scalar, ch)
		t.column++
	}

	value := string(scalar)
	return Token{
		Type:   TokenPlainScalar,
		Value:  strings.TrimSpace(value),
		Line:   t.line,
		Column: t.column - len(value),
	}, nil
}

// NextToken gets next token.
func (t *Tokenizer) NextToken() (Token, error) {
	for {
		ch, _, err := t.reader.ReadRune()

		const debug = false
		if debug {
			fmt.Printf("%s: Read rune: %d, err: %v\n", statusName[t.status], ch, err)
		}

		if err == io.EOF {
			return t.returnEOF()
		}
		if err != nil {
			return Token{}, err
		}

		t.column++

		switch t.status {

		case statusBlank:

			switch ch {
			case '\n':
				return t.returnNewLine()
			case '-':
				// Only match dash if it's at the beginning of a line
				if t.column == 1 {
					ch, _, err := t.reader.ReadRune()
					if err == io.EOF {
						return t.returnDash()
					}

					/*
						if ch == '-' {
							// hit --, check for ---
							ch, _, err := t.reader.ReadRune()
							if err == io.EOF {
								return Token{
									Type:   TokenPlainScalar,
									Value:  "--",
									Line:   t.line,
									Column: t.column - 2,
								}, nil
							}
							if err != nil {
								t.reader.UnreadRune()
							} else {
								if ch == '-' {
									return Token{Type: TokenDocStart, Value: "---", Line: t.line, Column: t.column}, nil
								}
							}
						}
					*/

					if ch == ' ' || ch == '\n' {
						return t.unreadAndReturnDash()
					}
					if err := t.reader.UnreadRune(); err != nil {
						return Token{}, err
					}

					//t.status = statusBlank

					//var scalar []rune
					scalar := []rune{'-'}

					return t.collectPlainScalar(scalar)
				}
			}

		case statusAfterDash:
			t.status = statusBlank

			var scalar []rune

			// skip blanks
			for {
				if ch == '\n' {
					return t.returnNewLine()
				}
				if ch != ' ' && ch != '\t' {
					scalar = append(scalar, ch)
					break
				}

				ch, _, err = t.reader.ReadRune()
				if err == io.EOF {
					return t.returnEOF()
				}
				if err != nil {
					return Token{}, err
				}

				t.column++
			}

			// collect plain scalar

			return t.collectPlainScalar(scalar)

		default:
			return Token{}, fmt.Errorf("unexpected token status: %d", t.status)
		}

	}
}

// TokenType defines token type.
type TokenType int

// Token types.
const (
	TokenEOF TokenType = iota
	TokenDash
	TokenPlainScalar
	TokenNewLine
	TokenDocStart // for '---'
	TokenDocEnd   // for '...'
)

var tokenTypeName = []string{
	"EOF",
	"DASH",
	"PLAIN-SCALAR",
	"NEWLINE",
	"DOC-START",
	"DOC-END",
}

// TokenEqual checks two tokens for equality.
func TokenEqual(t1, t2 Token) bool {
	if t1.Type != t2.Type {
		return false
	}
	switch t1.Type {
	case TokenPlainScalar:
		return t1.Value == t2.Value
	}
	return true
}

// Token defines yaml token.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

func (t *Token) String() string {
	if t.Type == TokenPlainScalar {
		return tokenTypeName[t.Type] + "(" + t.Value + ")"
	}
	return tokenTypeName[t.Type] + "()"
}
