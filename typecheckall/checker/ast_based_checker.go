package checker

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

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

func parseFile(fset *token.FileSet, filename string, content string) (*ast.File, error) {
	f, err := parser.ParseFile(fset, filename, content, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func getZeroReturnFunctionBody(fn *ast.FuncDecl) *ast.BlockStmt {
	if fn.Type.Results == nil {
		return &ast.BlockStmt{List: nil}
	}

	var results []ast.Expr
	for _, field := range fn.Type.Results.List {
		// Dereference of the `new` function to return zero value. Even works for interfaces.
		// return *new(Type)
		results = append(results, &ast.StarExpr{
			X: &ast.CallExpr{
				Fun:  &ast.Ident{Name: "new"},
				Args: []ast.Expr{field.Type},
			},
		})
	}

	return &ast.BlockStmt{List: []ast.Stmt{
		&ast.ReturnStmt{Results: results},
	}}
}

func removeError(e types.Error, f *ast.File) {
	nodes, exact := astutil.PathEnclosingInterval(f, e.Pos, e.Pos)
	if !exact || len(nodes) == 0 {
		return
	}

	start := nodes[0].Pos()
	end := nodes[0].End()

	var function *ast.FuncDecl
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Pos() <= start && fn.End() >= end {
				function = fn
				break
			}
		}
	}

	if function == nil {
		return
	}
	function.Body = getZeroReturnFunctionBody(function)
}

type file struct {
	f       *ast.File
	name    string
	content string
}

// ASTBasedChecker implements a Checker that uses AST manipulation to remove function bodies
// and replace them with return statements that return zero values in the AST. This is more robust
// than TextBasedChecker and works on all valid Go code.
type ASTBasedChecker struct {
	files []file
	fset  *token.FileSet
}

func NewASTBasedChecker() Checker {
	return &ASTBasedChecker{
		files: []file{},
		fset:  token.NewFileSet(),
	}
}

func (c *ASTBasedChecker) AddFile(filename string, content string) error {
	f, err := parseFile(c.fset, filename, content)
	if err != nil {
		return err
	}

	c.files = append(c.files, file{f: f, name: filename, content: content})
	return nil
}

func (c *ASTBasedChecker) removeError(e types.Error) {
	for i := range c.files {
		removeError(e, c.files[i].f)
	}
}

func (c *ASTBasedChecker) checkFiles() *types.Error {
	conf := types.Config{
		Importer:                 importer.For("source", nil),
		DisableUnusedImportCheck: true,
	}

	files := make([]*ast.File, len(c.files))
	for i := range c.files {
		files[i] = c.files[i].f
	}

	_, err := conf.Check("pkg", c.fset, files, nil)
	if err == nil {
		return nil
	}

	typeErr, ok := err.(types.Error)
	if !ok {
		panic(err)
	}

	return &typeErr
}

func (c *ASTBasedChecker) getErrorFunction(e types.Error) (*ast.FuncDecl, error) {
	for i := range c.files {
		nodes, exact := astutil.PathEnclosingInterval(c.files[i].f, e.Pos, e.Pos)
		if !exact || len(nodes) == 0 {
			continue
		}

		exprNode := nodes[0]
		function := getParentFunc(c.files[i].f, exprNode)
		if function != nil {
			return function, nil
		}
	}

	return nil, fmt.Errorf("could not find parent function for error position (%w)", e)
}

func (c *ASTBasedChecker) collectErrors() (map[string]relativeFuncError, error) {
	funcErrors := make(map[string]relativeFuncError)

	for {
		typeErr := c.checkFiles()
		if typeErr == nil {
			// No more errors
			break
		}

		function, err := c.getErrorFunction(*typeErr)
		if err != nil {
			return funcErrors, err
		}

		funcError := toError(*typeErr, function, c.fset)

		// Early exit if we've already processed this function, for example if the error is in the function signature
		// which can't be removed.
		if _, exists := funcErrors[funcError.FuncName]; exists {
			fmt.Printf("Already processed function %s breaking to avoid infinite loop", funcError.FuncName)
			break
		}

		funcErrors[funcError.FuncName] = funcError

		c.removeError(*typeErr)
	}

	return funcErrors, nil
}

func (c *ASTBasedChecker) resolveErrors(errors map[string]relativeFuncError) ([]Error, error) {
	var resolved []Error

	asts := make(map[string]*ast.File)
	fset := token.NewFileSet()

	// Re-parse files to get original ASTs for position resolution
	for _, file := range c.files {
		f, err := parseFile(fset, file.name, file.content)
		if err != nil {
			return nil, err
		}
		asts[file.name] = f
	}

	for _, e := range errors {
		f, exists := asts[e.File]
		if !exists {
			return nil, fmt.Errorf("could not find file %s for error resolution", e.File)
		}

		function := getFunctionByName(f, e.FuncName)
		if function == nil {
			return nil, fmt.Errorf("could not find function %s in file %s for error resolution", e.FuncName, e.File)
		}

		functionPos := fset.Position(function.Pos())
		errorLine := functionPos.Line + e.RelLine - 1

		resolved = append(resolved, Error{
			FuncName: e.FuncName,
			File:     e.File,
			Line:     errorLine,
			Column:   e.Column,
			Msg:      e.Message,
		})
	}

	slices.SortFunc(resolved, func(a, b Error) int {
		if a.File != b.File {
			return strings.Compare(a.File, b.File)
		}
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		return a.Column - b.Column
	})

	return resolved, nil
}

func (c *ASTBasedChecker) Check() ([]Error, error) {
	funcErrors, err := c.collectErrors()
	if err != nil {
		return nil, err
	}

	resolved, err := c.resolveErrors(funcErrors)
	if err != nil {
		return nil, err
	}

	return resolved, nil
}
