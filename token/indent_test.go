package token

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
)

var indentTestTable = []tokenizerTest{
	{"dash with indent", " -", []Token{
		{Type: TokenIndent}, {Type: TokenDash}, {Type: TokenDedent}}},
}

// go test -count 1 -run '^TestIndent$' ./...
func TestIndent(t *testing.T) {

	debug := isDebugEnabled()

	for i, data := range indentTestTable {
		name := fmt.Sprintf("%02d of %02d: %s", i+1, len(indentTestTable), data.name)

		t.Run(name, func(t *testing.T) {
			tokenizer := NewTokenizer(strings.NewReader(data.input), debug)
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

			if !slices.EqualFunc(data.expected, tokens, TokenEqual) {
				t.Errorf("wrong:\nexpected:%v\n     got:%v",
					formatTokens(data.expected), formatTokens(tokens))
			}
		})

	}
}
