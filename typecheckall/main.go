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

func mockFunctionBody(fn *ast.FuncDecl) *ast.BlockStmt {
	// Create a mock body that just returns zero values
	var stmts []ast.Stmt

	// Create return statement with zero values
	if fn.Type.Results == nil {
		return &ast.BlockStmt{List: stmts}
	}

	var results []ast.Expr
	for _, field := range fn.Type.Results.List {

		results = append(results, &ast.StarExpr{
			X: &ast.CallExpr{
				Fun:  &ast.Ident{Name: "new"},
				Args: []ast.Expr{field.Type},
			},
		})

		//var zeroValue ast.Expr
		//switch t := field.Type.(type) {
		//case *ast.Ident:
		//	switch t.Name {
		//	case "int", "float64", "byte", "rune":
		//		zeroValue = &ast.BasicLit{Kind: token.INT, Value: "0"}
		//	case "string":
		//		zeroValue = &ast.BasicLit{Kind: token.STRING, Value: `""`}
		//	case "bool":
		//		zeroValue = &ast.Ident{Name: "false"}
		//	default:
		//		zeroValue = &ast.Ident{Name: "nil"}
		//	}
		//default:
		//	zeroValue = &ast.Ident{Name: "nil"}
		//}
		//results = append(results, zeroValue)
	}
	stmts = append(stmts, &ast.ReturnStmt{Results: results})

	return &ast.BlockStmt{List: stmts}
}

func removeError(file *ast.File, e types.Error) *ast.File {
	//lines := strings.Split(prog, "\n")

	nodes, exact := astutil.PathEnclosingInterval(file, e.Pos, e.Pos)
	if !exact || len(nodes) == 0 {
		return nil
	}

	start := nodes[0].Pos()
	end := nodes[0].End()

	var function *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Pos() <= start && fn.End() >= end {
				function = fn
				break
			}
		}
	}

	if function == nil {
		return nil
	}
	//var function *ast.FuncDecl
	//ast.Inspect(file, func(n ast.Node) bool {
	//	if n, ok := n.(*ast.FuncDecl); ok {
	//		function = n
	//	}
	//
	//	if n != nil && n.Pos() > start {
	//		return false
	//	}
	//
	//	//if n == nil {
	//	//	// Done with node's children. Pop.
	//	//	stack = stack[:len(stack)-1]
	//	//} else {
	//	//	// Push the current node for children.
	//	//	stack = append(stack, n)
	//	//}
	//
	//	return true
	//})
	mockBody := mockFunctionBody(function)
	function.Body = mockBody

	fmt.Printf("Removing function %s due to error: %s\n", function.Name.Name, e.Msg)
	return file
}

func getErrors(prog string, name string, fset *token.FileSet) []types.Error {
	f, err := parser.ParseFile(fset, name, prog, parser.AllErrors)
	if err != nil {
		panic(err)
	}

	conf := types.Config{
		Importer: importer.Default(),
	}

	pkg := types.NewPackage("main", "")

	info := &types.Info{}
	checker := types.NewChecker(&conf, fset, pkg, info)

	var errors = []types.Error{}
	var astFile = f
	for astFile != nil {
		err = checker.Files([]*ast.File{astFile})
		if err == nil {
			return nil
		}

		typeErr, ok := err.(types.Error)
		if !ok {
			panic(err)
		}
		errors = append(errors, typeErr)

		astFile = removeError(f, typeErr)
	}

	return errors

	//if removedProg == "" {
	//	return errors
	//}
	//
	//errors = append(errors, getErrors(removedProg)...)
	//return errors
}

func checkFile(filename string) {
	contentRaw, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	errors := checkProg(string(contentRaw), filename)
	for _, e := range errors {
		fmt.Printf("Type error in function %s at line %d, column %d\n", e.FuncName, e.RelLine, e.Column)
	}
	//fset := token.NewFileSet()
	//files := map[string]*sourceFile{}
	//
	//file, err := parseFile(files, fset, filename)
	//if err != nil {
	//	panic(err)
	//}
	//
	//errors := getErrors(file.content, filename, fset)
	//for _, e := range errors {
	//	println("\n-------------------------\n")
	//	handleTypeError(e, files, fset)
	//}
}

const errorHere = "Here"
const ansiiRed = "\033[31m"
const ansiiOrange = "\033[33m"
const ansiiReset = "\033[0m"
const ansiiBold = "\033[1m"

const extraContextLines = 3

type FuncError struct {
	FuncName string
	RelLine  int
	Column   int
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

func getError(f *ast.File, fset *token.FileSet) *types.Error {
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
	}
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

func removeErrorFromProg(prog string, fn *ast.FuncDecl, fset *token.FileSet) string {
	functionPos := fset.Position(fn.Pos())
	functionPosEnd := fset.Position(fn.End())

	lines := strings.Split(prog, "\n")
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

func checkProg(prog string, name string) []FuncError {
	errors := make(map[string]FuncError)

	for {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, prog, parser.AllErrors)
		if err != nil {
			panic(err)
		}

		err = getError(f, fset)
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

		// Save temporary file
		//tempFilename := "temp_fixed.gotest"
		//err = os.WriteFile(tempFilename, []byte(prog), 0644)
		//if err != nil {
		//	panic(err)
		//}

		funcError := toFuncError(*typeErr, function, fset)

		if _, exists := errors[funcError.FuncName]; exists {
			println("Already processed function", funcError.FuncName, "breaking to avoid infinite loop")
			break
		}

		errors[funcError.FuncName] = funcError

		prog = removeErrorFromProg(prog, function, fset)
	}

	var result []FuncError
	for v := range maps.Values(errors) {
		result = append(result, v)
	}

	return result
}

func handleTypeError(e types.Error, files map[string]*sourceFile, fset *token.FileSet) {
	pos := fset.Position(e.Pos)
	file, ok := files[pos.Filename]
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

func main() {
	if len(os.Args) < 2 {
		panic("expected at least one filename as argument")
	}

	for _, filename := range os.Args[1:] {
		checkFile(filename)
	}
}
