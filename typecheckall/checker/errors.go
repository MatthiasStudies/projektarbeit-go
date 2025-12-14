package checker

import (
	"go/ast"
	"go/token"
	"go/types"
)

type relativeFuncError struct {
	FuncName string
	File     string
	RelLine  int
	Column   int
	Message  string
}

type Error struct {
	FuncName string
	File     string
	Line     int
	Column   int
	Msg      string
}

func toError(e types.Error, function *ast.FuncDecl, fset *token.FileSet) relativeFuncError {
	functionPos := fset.Position(function.Pos())
	errorPos := fset.Position(e.Pos)

	relLine := errorPos.Line - functionPos.Line + 1
	column := errorPos.Column

	return relativeFuncError{
		FuncName: function.Name.Name,
		File:     functionPos.Filename,
		RelLine:  relLine,
		Column:   column,
		Message:  e.Msg,
	}
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
