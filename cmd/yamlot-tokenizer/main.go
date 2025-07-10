// Package main implements the tool.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/udhos/yamlot/token"
)

func main() {
	tokenizer := token.NewTokenizer(os.Stdin)
	for {
		t, err := tokenizer.NextToken()
		if err == io.EOF && t.Type == token.TokenEOF {
			break
		}
		if err != nil {
			fmt.Printf("error: %v", err)
			break
		}
		fmt.Printf("line=%03d column=%02d: %s\n", t.Line, t.Column, t.String())
	}
}
