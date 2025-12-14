package main

import (
	"fmt"
	"os"
	"strings"

	"typecheckall/checker"
)

func showError(column int) {
	for i := 1; i < column; i++ {
		fmt.Print(" ")
	}
	fmt.Println("^")
}

func showErrors(code string, errors []checker.Error) {
	lines := strings.Split(code, "\n")
	for _, e := range errors {
		fmt.Printf("Type error in function `%s` at line %d, column %d: %s\n", e.FuncName, e.Line, e.Column, e.Msg)
		if e.Line-1 >= 0 && e.Line-1 < len(lines) {
			fmt.Println(lines[e.Line-1])
			showError(e.Column)
		}
	}
}

func checkFile(filename string) {
	contentRaw, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	content := strings.ReplaceAll(string(contentRaw), "\t", "  ")

	c, err := checker.NewASTBasedChecker(content)
	//c, err := checker.NewTextBasedChecker(content)
	if err != nil {
		panic(err)
	}

	errors, err := c.Check()
	if err != nil {
		panic(err)
	}

	// do something with the errors
	showErrors(content, errors)
}

func main() {
	if len(os.Args) < 2 {
		panic("expected at least one filename as argument")
	}

	for _, filename := range os.Args[1:] {
		checkFile(filename)
	}
}
