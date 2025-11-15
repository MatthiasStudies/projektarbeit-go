package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
)

const inspectPrefix = "inspect:"

func findLookupNames(commentText string) []string {
	if !strings.HasPrefix(commentText, inspectPrefix) {
		return nil
	}
	text := strings.TrimPrefix(commentText, inspectPrefix)
	parts := strings.Split(text, ",")
	var names []string
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func formatObj(fset *token.FileSet, obj types.Object) string {
	if obj == nil {
		return "\t<not found>\n"
	}
	pos := fset.Position(obj.Pos())

	buff := &strings.Builder{}
	fmt.Fprintf(buff, "\tKind: %T\n", obj)
	fmt.Fprintf(buff, "\tType: %s\n", obj.Type().String())
	fmt.Fprintf(buff, "\tPkg: %v\n", obj.Pkg())
	fmt.Fprintf(buff, "\tPos: %v\n", pos)
	if v, ok := obj.(*types.Var); ok {
		fmt.Fprintf(buff, "\tVar isExported: %v\n", v.Exported())
	}
	if f, ok := obj.(*types.Func); ok {
		sig := f.Type().(*types.Signature)
		fmt.Fprintf(buff, "\tFunc Params: %s\n", sig.Params().String())
		fmt.Fprintf(buff, "\tFunc Results: %s\n", sig.Results().String())
	}
	underlying := obj.Type().Underlying()
	fmt.Fprintf(buff, "\tUnderlying Type: %T %s\n", underlying, underlying.String())
	return buff.String()
}

func printObj(fset *token.FileSet, pos token.Pos, name string, obj types.Object) {
	fmt.Printf("%s,\t%q\n", fset.Position(pos), name)
	fmt.Println(formatObj(fset, obj))
}

func inspectCode(code string, fileName string) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, fileName, code, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	conf := types.Config{
		Importer: importer.Default(),
	}

	pkg := types.NewPackage("main", "")

	info := &types.Info{
		// Types: make(map[ast.Expr]types.TypeAndValue)
	}
	checker := types.NewChecker(&conf, fset, pkg, info)

	err = checker.Files([]*ast.File{f})
	if err != nil {
		panic(err)
	}

	for _, comment := range f.Comments {
		names := findLookupNames(comment.Text())
		if names == nil {
			continue
		}

		pos := comment.Pos()
		scope := pkg.Scope().Innermost(pos) // Find the scope closest to the comment position

		for _, name := range names {
			_, obj := scope.LookupParent(name, pos)
			printObj(fset, pos, name, obj)
		}
	}
}

func inspectFile(file string) {
	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	filename := filepath.Base(file)
	inspectCode(string(data), filename)
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("usage: inspect <file.go>")
		return
	}

	for _, file := range os.Args[1:] {
		fmt.Printf("Inspecting file: %s\n", file)
		inspectFile(file)
	}
}
