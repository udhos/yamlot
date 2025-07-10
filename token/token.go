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
	if err := t.reader.UnreadRune(); err != nil {
		return Token{}, err
	}
	tk := Token{Type: TokenDash, Value: "-", Line: t.line, Column: t.column}
	t.status = statusAfterDash
	return tk, nil
}

// NextToken gets next token.
func (t *Tokenizer) NextToken() (Token, error) {
	for {
		ch, _, err := t.reader.ReadRune()
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
					return t.returnDash()
				}
			}

		case statusAfterDash:
			t.status = statusBlank

			var scalar []rune

			// skip blanks
			for {
				ch, _, err := t.reader.ReadRune()
				if err == io.EOF {
					return t.returnEOF()
				}
				if err != nil {
					return Token{}, err
				}

				t.column++
				if ch == '\n' {
					return t.returnNewLine()
				}
				if ch != ' ' && ch != '\t' {
					scalar = append(scalar, ch)
					break
				}
			}

			// collect plain scalar

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
)

var tokenTypeName = []string{
	"EOF",
	"DASH",
	"PLAIN-SCALAR",
	"NEWLINE",
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
