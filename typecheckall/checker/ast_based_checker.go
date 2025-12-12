package checker

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ast/astutil"
)

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

// ASTBasedChecker implements a Checker that uses AST manipulation to remove function bodies
// and replace them with return statements that return zero values in the AST. This is more robust
// than TextBasedChecker and works on all valid Go code.
type ASTBasedChecker struct {
	code string
	f    *ast.File
	fset *token.FileSet
}

func NewASTBasedChecker(code string) (Checker, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "file.go", code, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	return &ASTBasedChecker{
		code: code,
		f:    f,
		fset: fset,
	}, nil
}

func (c *ASTBasedChecker) removeError(e types.Error) {
	nodes, exact := astutil.PathEnclosingInterval(c.f, e.Pos, e.Pos)
	if !exact || len(nodes) == 0 {
		return
	}

	start := nodes[0].Pos()
	end := nodes[0].End()

	var function *ast.FuncDecl
	for _, decl := range c.f.Decls {
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

func (c *ASTBasedChecker) collectErrors() (map[string]relativeFuncError, error) {
	funcErrors := make(map[string]relativeFuncError)

	for {
		typeErr := checkASTFile(c.f, c.fset)
		if typeErr == nil {
			// No more errors
			break
		}

		nodes, exact := astutil.PathEnclosingInterval(c.f, typeErr.Pos, typeErr.Pos)
		if !exact || len(nodes) == 0 {
			return funcErrors, errors.New("could not find AST node for error position")
		}

		exprNode := nodes[0]
		function := getParentFunc(c.f, exprNode)
		if function == nil {
			return funcErrors, errors.New("could not find parent function for error position")
		}

		funcError := toFuncError(*typeErr, function, c.fset)

		// Early exit if we've already processed this function, for example if the error is in the function signature
		// which can't be removed.
		if _, exists := funcErrors[funcError.FuncName]; exists {
			println("Already processed function", funcError.FuncName, "breaking to avoid infinite loop")
			break
		}

		funcErrors[funcError.FuncName] = funcError

		c.removeError(*typeErr)
	}

	return funcErrors, nil
}

func (c *ASTBasedChecker) Check() ([]Error, error) {
	funcErrors, err := c.collectErrors()
	if err != nil {
		return nil, err
	}

	resolved := resolveErrors(c.code, funcErrors)
	return resolved, nil
}
