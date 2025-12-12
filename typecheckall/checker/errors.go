package checker

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
)

type relativeFuncError struct {
	FuncName string
	RelLine  int
	Column   int
	Message  string
}

type Error struct {
	FuncName string
	Line     int
	Column   int
	Msg      string
}

func toFuncError(e types.Error, function *ast.FuncDecl, fset *token.FileSet) relativeFuncError {
	functionPos := fset.Position(function.Pos())
	errorPos := fset.Position(e.Pos)

	relLine := errorPos.Line - functionPos.Line + 1
	column := errorPos.Column

	return relativeFuncError{
		FuncName: function.Name.Name,
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

func resolveErrors(code string, errors map[string]relativeFuncError) []Error {
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

	slices.SortFunc(resolved, func(a, b Error) int {
		if a.Line != b.Line {
			return a.Line - b.Line
		}
		return a.Column - b.Column
	})

	return resolved
}
