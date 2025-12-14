package checker

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

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

// TextBasedChecker implements a Checker that mocks functions by removing their bodies
// and replacing them with return statements that return zero values in the source code text. This does only work
// on well-formatted code.
type TextBasedChecker struct {
	code string
}

func NewTextBasedChecker(file string) (Checker, error) {
	return &TextBasedChecker{
		code: file,
	}, nil
}

func (c *TextBasedChecker) removeFunctionFromCode(fn *ast.FuncDecl, fset *token.FileSet) error {
	functionPos := fset.Position(fn.Pos())
	functionPosEnd := fset.Position(fn.End())

	lines := strings.Split(c.code, "\n")
	if functionPos.Line-1 < 0 || functionPos.Line-1 >= len(lines) {
		return fmt.Errorf("function position out of bounds")
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

	c.code = strings.Join(lines, "\n")
	return nil
}

func (c *TextBasedChecker) collectErrors() (map[string]relativeFuncError, error) {
	funcErrors := make(map[string]relativeFuncError)

	for {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "file.go", c.code, parser.AllErrors)
		if err != nil {
			return nil, err
		}

		typeErr := checkASTFile(f, fset)
		if typeErr == nil {
			// No more errors
			break
		}

		nodes, exact := astutil.PathEnclosingInterval(f, typeErr.Pos, typeErr.Pos)
		if !exact || len(nodes) == 0 {
			return nil, fmt.Errorf("could not find AST node for error position")
		}

		exprNode := nodes[0]
		function := getParentFunc(f, exprNode)
		if function == nil {
			return nil, fmt.Errorf("could not find parent function for error position")
		}

		funcError := toError(*typeErr, function, fset)

		// Early exit if we've already processed this function, for example if the error is in the function signature
		// which can't be removed.
		if _, exists := funcErrors[funcError.FuncName]; exists {
			println("Already processed function", funcError.FuncName, "breaking to avoid infinite loop")
			break
		}

		funcErrors[funcError.FuncName] = funcError

		err = c.removeFunctionFromCode(function, fset)
		if err != nil {
			return nil, err
		}
	}

	return funcErrors, nil
}

func (c *TextBasedChecker) Check() ([]Error, error) {
	funcErrors, err := c.collectErrors()
	if err != nil {
		return nil, err
	}

	resolved := resolveErrors(c.code, funcErrors)
	return resolved, nil
}
