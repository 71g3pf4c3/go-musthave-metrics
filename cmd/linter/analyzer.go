package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer reports:
//   - any use of the built-in panic function
//   - calls to log.Fatal* / os.Exit outside func main of package main
var Analyzer = &analysis.Analyzer{
	Name: "noexit",
	Doc:  "reports usage of panic, and log.Fatal/os.Exit outside func main of package main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		pkgName := file.Name.Name

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			switch fn := call.Fun.(type) {
			case *ast.Ident:
				// Built-in panic().
				if fn.Name == "panic" {
					pass.Reportf(call.Pos(), "avoid using panic")
				}
			case *ast.SelectorExpr:
				ident, ok := fn.X.(*ast.Ident)
				if !ok {
					return true
				}

				isLogFatal := ident.Name == "log" && (fn.Sel.Name == "Fatal" || fn.Sel.Name == "Fatalf" || fn.Sel.Name == "Fatalln")
				isOsExit := ident.Name == "os" && fn.Sel.Name == "Exit"

				if isLogFatal || isOsExit {
					if !isInsideMainFunc(call, file, pkgName) {
						pass.Reportf(call.Pos(), "avoid calling %s.%s outside func main of package main", ident.Name, fn.Sel.Name)
					}
				}
			}

			return true
		})
	}
	return nil, nil
}

// isInsideMainFunc checks whether pos falls inside func main() of package main.
func isInsideMainFunc(node ast.Node, file *ast.File, pkgName string) bool {
	if pkgName != "main" {
		return false
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" || fn.Recv != nil {
			continue
		}
		if node.Pos() >= fn.Body.Pos() && node.End() <= fn.Body.End() {
			return true
		}
	}
	return false
}
