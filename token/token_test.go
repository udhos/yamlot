package token

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestTokenizerEmpty(t *testing.T) {
	tokenizer := NewTokenizer(strings.NewReader(""))
	expectEOF(t, tokenizer)
}

func expectEOF(t *testing.T, tokenizer *Tokenizer) {
	tk, err := tokenizer.NextToken()
	if err != io.EOF {
		t.Errorf("expecting EOF error, got: %v", err)
	}
	if tk.Type != TokenEOF {
		t.Errorf("expecting EOF token, got: %v", tk)
	}
}

func TestTokenizerLines(t *testing.T) {
	tokenizer := NewTokenizer(strings.NewReader("\n\n"))
	{
		tk, err := tokenizer.NextToken()
		if err != nil {
			t.Error(err)
		}
		if tk.Type != TokenNewLine {
			t.Errorf("expecting newline 1, got: %v", tk)
		}
	}

	tk, err := tokenizer.NextToken()
	if err != nil {
		t.Error(err)
	}
	if tk.Type != TokenNewLine {
		t.Errorf("expecting newline 2, got: %v", tk)
	}

	expectEOF(t, tokenizer)
}

const simpleBlockSequence = `
- apple
- banana
- cherry
`

type tokenizerTest struct {
	name     string
	input    string
	expected []Token
}

var tokenizerTestTable = []tokenizerTest{
	{"dash", "\n-\n", []Token{{Type: TokenNewLine}, {Type: TokenDash}, {Type: TokenNewLine}}},
	{"sequence", simpleBlockSequence,
		[]Token{
			{Type: TokenNewLine},
			{Type: TokenDash}, {Type: TokenPlainScalar, Value: "apple"}, {Type: TokenNewLine},
			{Type: TokenDash}, {Type: TokenPlainScalar, Value: "banana"}, {Type: TokenNewLine},
			{Type: TokenDash}, {Type: TokenPlainScalar, Value: "cherry"}, {Type: TokenNewLine},
		},
	},
}

func TestTokenizer(t *testing.T) {
	for i, data := range tokenizerTestTable {
		name := fmt.Sprintf("%02d of %02d: %s", i+1, len(tokenizerTestTable), data.name)

		t.Run(name, func(t *testing.T) {
			tokenizer := NewTokenizer(strings.NewReader(data.input))
			var tokens []Token
			for {
				tk, err := tokenizer.NextToken()
				if err == io.EOF && tk.Type == TokenEOF {
					break
				}
				if err != nil {
					t.Error(err)
					return
				}
				tokens = append(tokens, tk)
			}

			if len(data.expected) != len(tokens) {
				t.Errorf("wrong length: expected=%d got=%d\nexpected:%v\n     got:%v",
					len(data.expected), len(tokens),
					formatTokens(data.expected), formatTokens(tokens))
				return
			}

			if !slices.EqualFunc(data.expected, tokens, TokenEqual) {
				t.Errorf("wrong:\nexpected:%v\n     got:%v",
					formatTokens(data.expected), formatTokens(tokens))
			}
		})

	}
}

func formatTokens(list []Token) string {
	var result []string
	for _, t := range list {
		result = append(result, t.String())
	}
	return strings.Join(result, "")
}
