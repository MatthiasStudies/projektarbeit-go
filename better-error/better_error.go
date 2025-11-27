package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

const errorHere = "Here"
const ansiiRed = "\033[31m"
const ansiiOrange = "\033[33m"
const ansiiReset = "\033[0m"
const ansiiBold = "\033[1m"

const extraContextLines = 10

func handleTypeError(e types.Error, fiels map[string]*sourceFile, fset *token.FileSet) {
	pos := fset.Position(e.Pos)
	file, ok := fiels[pos.Filename]
	if !ok {
		panic("file content not found")
	}

	fileLines := strings.Split(file.content, "\n")

	nodes, exact := astutil.PathEnclosingInterval(file.astFile, e.Pos, e.Pos)
	if !exact || len(nodes) == 0 {
		panic("could not find AST node for error position")
	}

	var out strings.Builder
	fmt.Fprintf(&out, "Type error in %s at line %d, column %d:\n", pos.Filename, pos.Line, pos.Column)

	// Highlight the erroneous expression
	exprNode := nodes[0]
	errNodeStart := fset.Position(exprNode.Pos())
	errNodeEnd := fset.Position(exprNode.End())

	visibleStartLine := max(errNodeStart.Line-1-extraContextLines, 0)
	visibleEndline := min(errNodeEnd.Line+extraContextLines, len(fileLines))

	for i := visibleStartLine; i < visibleEndline; i++ {
		lineNum := i + 1
		lineContent := fileLines[i]

		var linePrefix string
		switch lineNum {
		case errNodeStart.Line:
			linePrefix = fmt.Sprintf("%s>%s ", ansiiRed, ansiiReset)
		case errNodeEnd.Line:
			linePrefix = fmt.Sprintf("%s<%s ", ansiiRed, ansiiReset)
		default:
			linePrefix = "  "
		}

		fmt.Fprintf(&out, "%s%4d | %s\n", linePrefix, lineNum, lineContent)

		switch lineNum {
		case errNodeStart.Line:
			markerStart := errNodeStart.Column - 1
			markerEnd := len(lineContent)
			if errNodeStart.Line == errNodeEnd.Line {
				markerEnd = errNodeEnd.Column - 1
			}
			fmt.Fprintf(&out, "       | %s%s%s%s%s%s\n",
				strings.Repeat(" ", markerStart),
				ansiiRed,
				strings.Repeat("^", markerEnd-markerStart),
				"---- Here",
				ansiiReset,
				"")
		case errNodeEnd.Line:
			markerEnd := errNodeEnd.Column - 1
			fmt.Fprintf(&out, "     | %s%s%s%s%s\n",
				strings.Repeat(" ", 0),
				ansiiRed,
				strings.Repeat("^", markerEnd),
				ansiiReset,
				"")
		}
	}

	fmt.Fprintf(&out, "%sError: %s%s%s\n", ansiiRed, ansiiOrange, e.Msg, ansiiReset)

	fmt.Print(out.String())
}

type sourceFile struct {
	content string
	astFile *ast.File
}

func parseFile(m map[string]*sourceFile, fset *token.FileSet, filename string) (*sourceFile, error) {
	contentRaw, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	content := strings.ReplaceAll(string(contentRaw), "\t", "    ")
	astFile, err := parser.ParseFile(fset, filename, content, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	f := &sourceFile{
		content: string(content),
		astFile: astFile,
	}
	m[filename] = f
	return f, nil
}

func checkFile(filename string) {
	files := map[string]*sourceFile{}
	// content, err := os.ReadFile(filename)
	// if err != nil {
	// 	panic(err)
	// }
	// files[filename] = string(content)

	fset := token.NewFileSet()

	f, err := parseFile(files, fset, filename)
	if err != nil {
		panic(err)
	}

	conf := types.Config{
		Importer: importer.Default(),
	}

	pkg := types.NewPackage("main", "")

	info := &types.Info{}
	checker := types.NewChecker(&conf, fset, pkg, info)

	err = checker.Files([]*ast.File{f.astFile})
	if err == nil {
		return
	}

	switch e := err.(type) {
	case types.Error:
		handleTypeError(e, files, fset)
	default:
		panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		panic("expected at least one filename as argument")
	}

	for _, filename := range os.Args[1:] {
		checkFile(filename)
	}
}
