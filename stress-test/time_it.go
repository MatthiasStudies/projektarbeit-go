package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"slices"
	"time"
)

const ITER_COUNT = 500

func timeTypechecking(f *ast.File, fset *token.FileSet) float64 {
	conf := types.Config{
		Importer: importer.Default(),
	}

	pkg := types.NewPackage("main", "")

	info := &types.Info{}
	checker := types.NewChecker(&conf, fset, pkg, info)
	start := time.Now()
	err := checker.Files([]*ast.File{f})
	if err != nil {
		panic(err)
	}
	elapsed := time.Since(start).Seconds() * 1000.0
	return elapsed
}

func timeFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, "main.go", content, parser.AllErrors)
	if err != nil {
		panic(err)
	}

	var times []float64

	for range ITER_COUNT {
		elapsed := timeTypechecking(f, fset)
		times = append(times, elapsed)
	}

	var total float64
	for _, t := range times {
		total += t
	}

	avg := total / float64(len(times))
	fmt.Println("File:", filename)
	fmt.Println("Average time over", ITER_COUNT, "iterations:", avg, "ms")

	min := slices.Min(times)
	fmt.Println("Minimum time:", min, "ms")

	max := slices.Max(times)
	fmt.Println("Maximum time:", max, "ms")
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		panic("expected at least one filename as argument")
	}

	for _, filename := range os.Args[1:] {
		timeFile(filename)
	}
}
