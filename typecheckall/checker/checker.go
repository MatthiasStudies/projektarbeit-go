package checker

import (
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
)

type Checker interface {
	Check() ([]Error, error)
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
