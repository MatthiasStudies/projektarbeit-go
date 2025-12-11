package main

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

const testCode = `
package main

type Interface interface {
	method()
}

type Struct struct {
	field int
}

func (s *Struct) method() {}
`

func main() {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "test.go", testCode, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	conf := types.Config{
		Importer: importer.Default(),
	}

	pkg := types.NewPackage("main", "")

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}
	checker := types.NewChecker(&conf, fset, pkg, info)

	err = checker.Files([]*ast.File{f})
	if err != nil {
		panic(err)
	}

	var structType = pkg.Scope().Lookup("Struct").Type().(*types.Named) // ensure it's a struct

	var interfaceType = pkg.Scope().Lookup("Interface").Type().Underlying()

	println("Assignable", types.AssignableTo(structType, interfaceType))
}
