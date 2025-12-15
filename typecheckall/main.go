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

func showErrors(errors []checker.Error, files map[string]string) {
	for _, e := range errors {
		fmt.Printf("Type error in function `%s` at %s:%d, column %d: %s\n", e.FuncName, e.File, e.Line, e.Column, e.Msg)
		if content, exists := files[e.File]; exists {
			lines := strings.Split(content, "\n")
			if e.Line-1 >= 0 && e.Line-1 < len(lines) {
				fmt.Println(lines[e.Line-1])
				showError(e.Column)
			}
		}
	}
}

func readFiles(filenames []string) map[string]string {
	files := make(map[string]string)

	for _, filename := range filenames {
		contentRaw, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		// Tabs are replaced for easier column calculation during error display. Not necessary for type checking.
		files[filename] = strings.ReplaceAll(string(contentRaw), "\t", "  ")
	}

	return files
}

func checkFiles(filenames []string) {
	files := readFiles(filenames)
	c := checker.NewASTBasedChecker()

	for name, content := range files {
		err := c.AddFile(name, content)
		if err != nil {
			panic(err)
		}
	}

	errors, err := c.Check()
	if err != nil {
		panic(err)
	}

	showErrors(errors, files)
}

func main() {
	if len(os.Args) < 2 {
		panic("usage: typecheckall <file1.go> <file2.go> ...")
	}

	checkFiles(os.Args[1:])
}
