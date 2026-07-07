package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	pkgs, err := scanPackages(root)
	if err != nil {
		log.Fatal(err)
	}

	for dir, structs := range pkgs {
		if len(structs) == 0 {
			continue
		}
		if err := generateFile(dir, structs); err != nil {
			log.Fatalf("generate %s: %v", dir, err)
		}
		fmt.Printf("generated %s/reset.gen.go (%d structs)\n", dir, len(structs))
	}
}

// structInfo holds information about a struct annotated with // generate:reset.
type structInfo struct {
	name   string
	fields []fieldInfo
}

// fieldInfo holds information about a single struct field.
type fieldInfo struct {
	name     string
	typExpr  ast.Expr
	typStr   string
	kind     fieldKind
	elemKind fieldKind // for pointers and slices — kind of the element
	elemStr  string    // string representation of element type (for pointers)
}

type fieldKind int

const (
	kindPrimitive fieldKind = iota
	kindString
	kindSlice
	kindMap
	kindStruct
	kindPointer
	kindInterface
	kindOther
)

// scanPackages walks directories starting from root and collects annotated structs grouped by directory.
func scanPackages(root string) (map[string][]structInfo, error) {
	result := make(map[string][]structInfo)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if base == "vendor" || base == "testdata" || base == ".git" {
			return filepath.SkipDir
		}

		structs, parseErr := parseDir(path)
		if parseErr != nil {
			return parseErr
		}
		if len(structs) > 0 {
			result[path] = structs
		}
		return nil
	})
	return result, err
}

// parseDir parses all Go files in dir and returns structs annotated with // generate:reset.
func parseDir(dir string) ([]structInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go") && fi.Name() != "reset.gen.go"
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var structs []structInfo

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				if !hasResetComment(gd.Doc) {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					si := structInfo{name: ts.Name.Name}
					for _, f := range st.Fields.List {
						fi := analyzeField(f)
						for _, name := range f.Names {
							fi2 := fi
							fi2.name = name.Name
							si.fields = append(si.fields, fi2)
						}
					}
					structs = append(structs, si)
				}
			}
		}
	}
	return structs, nil
}

// hasResetComment checks whether the doc comment group contains a line "// generate:reset".
func hasResetComment(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if strings.TrimSpace(c.Text) == "// generate:reset" {
			return true
		}
	}
	return false
}

// analyzeField determines the kind and string representation of a field's type.
func analyzeField(f *ast.Field) fieldInfo {
	fi := fieldInfo{
		typExpr: f.Type,
		typStr:  exprString(f.Type),
	}
	fi.kind = classifyExpr(f.Type)

	switch t := f.Type.(type) {
	case *ast.StarExpr:
		fi.elemKind = classifyExpr(t.X)
		fi.elemStr = exprString(t.X)
	case *ast.ArrayType:
		fi.elemKind = classifyExpr(t.Elt)
		fi.elemStr = exprString(t.Elt)
	}
	return fi
}

// classifyExpr determines the kind of a type expression.
func classifyExpr(expr ast.Expr) fieldKind {
	switch e := expr.(type) {
	case *ast.Ident:
		switch e.Name {
		case "bool", "byte", "rune",
			"int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"float32", "float64",
			"complex64", "complex128":
			return kindPrimitive
		case "string":
			return kindString
		default:
			return kindStruct
		}
	case *ast.StarExpr:
		return kindPointer
	case *ast.ArrayType:
		return kindSlice
	case *ast.MapType:
		return kindMap
	case *ast.SelectorExpr:
		return kindStruct
	case *ast.InterfaceType:
		return kindInterface
	default:
		return kindOther
	}
}

// exprString converts an ast.Expr to its Go source string.
func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + exprString(e.Elt)
		}
		return "[...]" + exprString(e.Elt)
	case *ast.MapType:
		return "map[" + exprString(e.Key) + "]" + exprString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// generateFile writes reset.gen.go in the given directory.
func generateFile(dir string, structs []structInfo) error {
	// Determine the package name from existing go files.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go") && fi.Name() != "reset.gen.go"
	}, parser.PackageClauseOnly)
	if err != nil {
		return err
	}
	var pkgName string
	for name := range pkgs {
		pkgName = name
		break
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by cmd/reset; DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n", pkgName)

	for _, s := range structs {
		receiver := strings.ToLower(s.name[:1])
		fmt.Fprintf(&buf, "\nfunc (%s *%s) Reset() {\n", receiver, s.name)
		fmt.Fprintf(&buf, "\tif %s == nil {\n\t\treturn\n\t}\n\n", receiver)

		for _, f := range s.fields {
			writeFieldReset(&buf, receiver, f)
		}
		fmt.Fprintf(&buf, "}\n")
	}

	path := filepath.Join(dir, "reset.gen.go")
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// writeFieldReset writes the reset statement for a single field.
func writeFieldReset(buf *bytes.Buffer, recv string, f fieldInfo) {
	accessor := recv + "." + f.name

	switch f.kind {
	case kindPrimitive:
		fmt.Fprintf(buf, "\t%s = 0\n", accessor)
	case kindString:
		fmt.Fprintf(buf, "\t%s = \"\"\n", accessor)
	case kindSlice:
		fmt.Fprintf(buf, "\t%s = %s[:0]\n", accessor, accessor)
	case kindMap:
		fmt.Fprintf(buf, "\tclear(%s)\n", accessor)
	case kindStruct:
		// If the struct implements Reset(), call it via interface assertion.
		fmt.Fprintf(buf, "\tif resetter, ok := interface{}(&%s).(interface{ Reset() }); ok {\n", accessor)
		fmt.Fprintf(buf, "\t\tresetter.Reset()\n")
		fmt.Fprintf(buf, "\t}\n")
	case kindPointer:
		fmt.Fprintf(buf, "\tif %s != nil {\n", accessor)
		writePointerDerefReset(buf, accessor, f)
		fmt.Fprintf(buf, "\t}\n")
	default:
		// Unknown/interface types — skip.
	}
}

// writePointerDerefReset writes the reset for a dereferenced pointer.
func writePointerDerefReset(buf *bytes.Buffer, accessor string, f fieldInfo) {
	switch f.elemKind {
	case kindPrimitive:
		fmt.Fprintf(buf, "\t\t*%s = 0\n", accessor)
	case kindString:
		fmt.Fprintf(buf, "\t\t*%s = \"\"\n", accessor)
	case kindSlice:
		fmt.Fprintf(buf, "\t\t*%s = (*%s)[:0]\n", accessor, accessor)
	case kindMap:
		fmt.Fprintf(buf, "\t\tclear(*%s)\n", accessor)
	case kindStruct:
		// Check if pointed-to struct has Reset().
		fmt.Fprintf(buf, "\t\tif resetter, ok := interface{}(%s).(interface{ Reset() }); ok {\n", accessor)
		fmt.Fprintf(buf, "\t\t\tresetter.Reset()\n")
		fmt.Fprintf(buf, "\t\t}\n")
	case kindPointer:
		// Nested pointer — just nil it.
		fmt.Fprintf(buf, "\t\t*%s = nil\n", accessor)
	default:
		// Unknown — skip.
	}
}
