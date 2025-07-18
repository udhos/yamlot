package token

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

func isDebugEnabled() bool {
	return strings.HasPrefix(strings.ToLower(os.Getenv("DEBUG")), "t")
}

func TestTokenizerEmpty(t *testing.T) {
	tokenizer := NewTokenizer(strings.NewReader(""), isDebugEnabled())
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

type errReader struct{}

func (r *errReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func TestTokenError(t *testing.T) {
	rd := &errReader{}
	tokenizer := NewTokenizer(rd, isDebugEnabled())
	tk, err := tokenizer.NextToken()
	if err == nil {
		t.Error("expecting error, got nil")
	}
	if tk.Type != TokenError {
		t.Errorf("expecting error token, got: %v", tk)
	}
}

func TestTokenizerLines(t *testing.T) {
	tokenizer := NewTokenizer(strings.NewReader("\n\n"), isDebugEnabled())
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
	{"dash-after-dash", "- - value\n", []Token{
		{Type: TokenDash, Line: 1, Column: 1},
		{Type: TokenPlainScalar, Value: "- value", Line: 1, Column: 3},
		{Type: TokenNewLine, Line: 1, Column: 10},
	}},
	{"double-dash-scalar-only", "--", []Token{
		{Type: TokenPlainScalar, Value: "--", Line: 1, Column: 1},
	}},
	{"double-dash-followed-by-text", "--a", []Token{
		{Type: TokenPlainScalar, Value: "--a", Line: 1, Column: 1},
	}},
	{"double-dash-followed-by-text-newline", "--a\n", []Token{
		{Type: TokenPlainScalar, Value: "--a", Line: 1, Column: 1},
		{Type: TokenNewLine, Line: 1, Column: 4},
	}},
	{"isolated-dash", "-", []Token{
		{Type: TokenDash, Line: 1, Column: 1},
	}},
	{"dash-followed-by-text", "-a", []Token{
		{Type: TokenPlainScalar, Value: "-a", Line: 1, Column: 1},
	}},
	{"double-dash-scalar-only-newline", "--\n", []Token{
		{Type: TokenPlainScalar, Value: "--", Line: 1, Column: 1}, {Type: TokenNewLine},
	}},
	{"isolated-dash-newline", "-\n", []Token{
		{Type: TokenDash, Line: 1, Column: 1}, {Type: TokenNewLine},
	}},
	{"dash-followed-by-text-newline", "-a\n", []Token{
		{Type: TokenPlainScalar, Value: "-a", Line: 1, Column: 1}, {Type: TokenNewLine},
	}},
	{"doc-start-marker", "---", []Token{
		{Type: TokenDocStart, Line: 1, Column: 1},
	}},
	{"doc-start-marker-newline", "---\n", []Token{
		{Type: TokenDocStart, Line: 1, Column: 1},
		{Type: TokenNewLine, Line: 1, Column: 4},
	}},
	{"doc-start-with-scalar", "--- hello\n", []Token{
		{Type: TokenDocStart, Line: 1, Column: 1},
		{Type: TokenPlainScalar, Value: "hello", Line: 1, Column: 5},
		{Type: TokenNewLine, Line: 1, Column: 10},
	}},
	{"doc-start-with-scalar-two-spaces", "---  hello\n", []Token{
		{Type: TokenDocStart, Line: 1, Column: 1},
		{Type: TokenPlainScalar, Value: "hello", Line: 1, Column: 5},
		{Type: TokenNewLine, Line: 1, Column: 10},
	}},
	{"false-doc-start-four-dashes", "----", []Token{
		{Type: TokenPlainScalar, Value: "----", Line: 1, Column: 1},
	}},
	{"false-doc-start-indented", "  ---", []Token{
		{Type: TokenPlainScalar, Value: "---", Line: 1, Column: 3},
	}},
	{"false-doc-start-inline", "---value", []Token{
		{Type: TokenPlainScalar, Value: "---value", Line: 1, Column: 1},
	}},
	{"scalar-with-tab", "-\tvalue\n", []Token{
		{Type: TokenDash, Line: 1, Column: 1},
		{Type: TokenPlainScalar, Value: "\tvalue", Line: 1, Column: 3},
		{Type: TokenNewLine, Line: 1, Column: 9},
	}},
	{"empty-scalar-two-spaces", "-  \n", []Token{
		{Type: TokenDash, Line: 1, Column: 1},
		{Type: TokenPlainScalar, Value: "", Line: 1, Column: 4},
		{Type: TokenNewLine, Line: 1, Column: 4},
	}},
	{"empty-scalar-after-dash", "- \n", []Token{
		{Type: TokenDash, Line: 1, Column: 1},
		{Type: TokenPlainScalar, Value: "", Line: 1, Column: 3},
		{Type: TokenNewLine, Line: 1, Column: 4},
	}},
	{"plain-scalar-with-spaces", "-  hello world  \n", []Token{
		{Type: TokenDash, Line: 1, Column: 1},
		{Type: TokenPlainScalar, Value: " hello world  ", Line: 1, Column: 3},
		{Type: TokenNewLine, Line: 1, Column: 17},
	}},
	{"plain-scalar-with-spaces-no-newline", "-  hello world  ", []Token{
		{Type: TokenDash, Line: 1, Column: 1},
		{Type: TokenPlainScalar, Value: " hello world  ", Line: 1, Column: 3},
	}},
}

// go test -count 1 -run '^TestTokenizer$' ./...
func TestTokenizer(t *testing.T) {
	for i, data := range tokenizerTestTable {
		name := fmt.Sprintf("%02d of %02d: %s", i+1, len(tokenizerTestTable), data.name)

		t.Run(name, func(t *testing.T) {
			tokenizer := NewTokenizer(strings.NewReader(data.input), isDebugEnabled())
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
	return strings.Join(result, ",")
}
