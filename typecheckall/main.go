package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type FuncError struct {
	FuncName string
	RelLine  int
	Column   int
	Message  string
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	default:
		return "unknown"
	}
}

func buildReturnZeroValues(fn *ast.FuncDecl) string {
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("return ")
	for i, field := range fn.Type.Results.List {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "*new(%s)", exprToString(field.Type))
	}
	return sb.String()
}

func removeFunctionFromCode(code string, fn *ast.FuncDecl, fset *token.FileSet) string {
	functionPos := fset.Position(fn.Pos())
	functionPosEnd := fset.Position(fn.End())

	lines := strings.Split(code, "\n")
	if functionPos.Line-1 < 0 || functionPos.Line-1 >= len(lines) {
		panic("function position out of range")
	}

	// Remove the function by replacing its lines with empty lines
	startLine := functionPos.Line - 1
	endLine := startLine + (functionPosEnd.Line - functionPos.Line)

	bodyStartLine := startLine + 1
	bodyEndLine := endLine - 1

	for i := bodyStartLine; i < bodyEndLine; i++ {
		lines[i] = ""
	}

	lines[bodyEndLine] = "    " + buildReturnZeroValues(fn)

	return strings.Join(lines, "\n")
}

func getParentFunc(f *ast.File, node ast.Node) *ast.FuncDecl {
	start := node.Pos()
	end := node.End()

	var function *ast.FuncDecl
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Pos() <= start && fn.End() >= end {
				function = fn
				break
			}
		}
	}
	return function
}

func checkASTFile(f *ast.File, fset *token.FileSet) *types.Error {
	conf := types.Config{
		Importer: importer.Default(),
	}

	pkg := types.NewPackage("main", "")

	info := &types.Info{}
	checker := types.NewChecker(&conf, fset, pkg, info)

	err := checker.Files([]*ast.File{f})
	if err == nil {
		return nil
	}

	typeErr, ok := err.(types.Error)
	if !ok {
		panic(err)
	}

	return &typeErr
}

func toFuncError(e types.Error, function *ast.FuncDecl, fset *token.FileSet) FuncError {
	functionPos := fset.Position(function.Pos())
	errorPos := fset.Position(e.Pos)

	relLine := errorPos.Line - functionPos.Line + 1
	column := errorPos.Column

	return FuncError{
		FuncName: function.Name.Name,
		RelLine:  relLine,
		Column:   column,
		Message:  e.Msg,
	}
}

func checkProgram(code string, name string) []FuncError {
	errors := make(map[string]FuncError)

	for {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, code, parser.AllErrors)
		if err != nil {
			panic(err)
		}

		err = checkASTFile(f, fset)
		if err == nil {
			break
		}

		typeErr, ok := err.(*types.Error)
		if !ok {
			panic(err)
		}
		if typeErr == nil {
			break
		}

		nodes, exact := astutil.PathEnclosingInterval(f, typeErr.Pos, typeErr.Pos)
		if !exact || len(nodes) == 0 {
			panic("could not find AST node for error position")
		}

		exprNode := nodes[0]
		function := getParentFunc(f, exprNode)
		if function == nil {
			panic("could not find parent function for error position")
		}

		funcError := toFuncError(*typeErr, function, fset)

		// Early exit if we've already processed this function, for example if the error is in the function signature
		// which can't be removed.
		if _, exists := errors[funcError.FuncName]; exists {
			println("Already processed function", funcError.FuncName, "breaking to avoid infinite loop")
			break
		}

		errors[funcError.FuncName] = funcError

		code = removeFunctionFromCode(code, function, fset)
	}

	var result []FuncError
	for v := range maps.Values(errors) {
		result = append(result, v)
	}

	return result
}

type Error struct {
	Line   int
	Column int
	Msg    string
}

func getFunctionByName(f *ast.File, name string) *ast.FuncDecl {
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == name {
				return fn
			}
		}
	}
	return nil
}

func resolveErrors(code string, errors []FuncError) []Error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "file.go", code, parser.AllErrors)
	if err != nil {
		panic(err)
	}

	var resolved []Error
	for _, e := range errors {
		fn := getFunctionByName(f, e.FuncName)
		if fn == nil {
			continue
		}
		functionPos := fset.Position(fn.Pos())
		resolved = append(resolved, Error{
			Line:   functionPos.Line + e.RelLine - 1,
			Column: e.Column,
			Msg:    e.Message,
		})
	}
	return resolved
}

func showError(line string, column int) {
	for i := 1; i < column; i++ {
		fmt.Print(" ")
	}
	fmt.Println("^")
}

func showErrors(code string, errors []Error) {
	lines := strings.Split(code, "\n")
	for _, e := range errors {
		fmt.Printf("Type error at line %d, column %d: %s\n", e.Line, e.Column, e.Msg)
		if e.Line-1 >= 0 && e.Line-1 < len(lines) {
			fmt.Println(lines[e.Line-1])
			showError(lines[e.Line-1], e.Column)
		}
	}
}

func checkFile(filename string) {
	contentRaw, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	content := strings.ReplaceAll(string(contentRaw), "\t", "  ")

	errors := checkProgram(content, filename)
	resovledErrors := resolveErrors(content, errors)
	showErrors(content, resovledErrors)
}

func main() {
	if len(os.Args) < 2 {
		panic("expected at least one filename as argument")
	}

	for _, filename := range os.Args[1:] {
		checkFile(filename)
	}
}
