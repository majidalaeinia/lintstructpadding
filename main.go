package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FieldInfo struct {
	Name     string
	Field    *ast.Field
	TypeSize int
}

func getTypeSize(expr ast.Expr) int {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "bool":
			return 1
		case "int8", "uint8", "byte":
			return 1
		case "int16", "uint16":
			return 2
		case "int32", "uint32", "rune", "float32":
			return 4
		case "int64", "uint64", "float64", "complex64":
			return 8
		case "complex128":
			return 16
		case "int", "uint", "uintptr":
			return 8
		case "string":
			return 16
		}
	case *ast.StarExpr:
		return 8
	case *ast.ArrayType:
		if t.Len == nil {
			return 24
		}
		return 8
	case *ast.MapType:
		return 8
	case *ast.ChanType:
		return 8
	case *ast.InterfaceType:
		return 16
	case *ast.FuncType:
		return 8
	}
	return 8
}

func analyzeStruct(structType *ast.StructType) ([]FieldInfo, bool) {
	if structType.Fields == nil || len(structType.Fields.List) <= 1 {
		return nil, false
	}
	var fields []FieldInfo

	for _, field := range structType.Fields.List {
		typeSize := getTypeSize(field.Type)

		if len(field.Names) > 0 {
			for _, name := range field.Names {
				fields = append(fields, FieldInfo{
					Field:    field,
					TypeSize: typeSize,
					Name:     name.Name,
				})
			}
		} else {
			fields = append(fields, FieldInfo{
				Field:    field,
				TypeSize: typeSize,
				Name:     "",
			})
		}
	}

	sortedFields := make([]FieldInfo, len(fields))
	copy(sortedFields, fields)

	sort.Slice(sortedFields, func(i, j int) bool {
		if sortedFields[i].TypeSize != sortedFields[j].TypeSize {
			return sortedFields[i].TypeSize > sortedFields[j].TypeSize
		}
		return sortedFields[i].Name < sortedFields[j].Name
	})

	needsReordering := false
	for i, field := range fields {
		if field.TypeSize != sortedFields[i].TypeSize || field.Name != sortedFields[i].Name {
			needsReordering = true
			break
		}
	}

	return sortedFields, needsReordering
}

func generateReorderedStruct(original *ast.StructType, reorderedFields []FieldInfo) *ast.StructType {
	newStruct := &ast.StructType{
		Struct: original.Struct,
		Fields: &ast.FieldList{
			Opening: original.Fields.Opening,
			Closing: original.Fields.Closing,
		},
	}

	fieldGroups := make(map[*ast.Field][]FieldInfo)
	for _, fi := range reorderedFields {
		fieldGroups[fi.Field] = append(fieldGroups[fi.Field], fi)
	}

	var newFields []*ast.Field
	processed := make(map[*ast.Field]bool)

	for _, fi := range reorderedFields {
		if processed[fi.Field] {
			continue
		}

		group := fieldGroups[fi.Field]
		if len(group) == 1 {
			newField := &ast.Field{
				Doc:     fi.Field.Doc,
				Names:   fi.Field.Names,
				Type:    fi.Field.Type,
				Tag:     fi.Field.Tag,
				Comment: fi.Field.Comment,
			}
			newFields = append(newFields, newField)
		} else {
			for _, f := range group {
				newField := &ast.Field{
					Doc:     f.Field.Doc,
					Type:    f.Field.Type,
					Tag:     f.Field.Tag,
					Comment: f.Field.Comment,
				}
				if f.Name != "" {
					newField.Names = []*ast.Ident{{Name: f.Name}}
				}
				newFields = append(newFields, newField)
			}
		}
		processed[fi.Field] = true
	}

	newStruct.Fields.List = newFields
	return newStruct
}

func fixStructsInFile(filename string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %v", err)
	}

	hasChanges := false

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							reorderedFields, needsReordering := analyzeStruct(structType)
							if needsReordering {
								hasChanges = true
								pos := fset.Position(typeSpec.Pos())
								fmt.Printf("Fixed struct '%s' at %s:%d:%d\n",
									typeSpec.Name.Name, filename, pos.Line, pos.Column)

								newStruct := generateReorderedStruct(structType, reorderedFields)
								typeSpec.Type = newStruct
							}
						}
					}
				}
			}
		}
		return true
	})

	if hasChanges {
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file: %v", err)
		}
		defer file.Close()

		if err := format.Node(file, fset, node); err != nil {
			return fmt.Errorf("failed to format code: %v", err)
		}

		fmt.Printf("Successfully fixed %s\n", filename)
	} else {
		fmt.Printf("✔ %s\n", filename)
	}

	return nil
}

func lintFile(filename string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %v", err)
	}

	hasIssues := false

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							reorderedFields, needsReordering := analyzeStruct(structType)
							if needsReordering {
								hasIssues = true
								pos := fset.Position(typeSpec.Pos())
								fmt.Printf("\n%s:%d:%d: struct '%s' fields can be reordered for better memory efficiency\n",
									filename, pos.Line, pos.Column, typeSpec.Name.Name)

								fmt.Println("Current order:")
								for _, field := range structType.Fields.List {
									if len(field.Names) > 0 {
										for _, name := range field.Names {
											size := getTypeSize(field.Type)
											fmt.Printf("  %s %s (size: %d bytes)\n",
												name.Name, formatType(field.Type), size)
										}
									} else {
										size := getTypeSize(field.Type)
										fmt.Printf("  %s (embedded, size: %d bytes)\n",
											formatType(field.Type), size)
									}
								}

								fmt.Println("Suggested order:")
								for _, fi := range reorderedFields {
									if fi.Name != "" {
										fmt.Printf("  %s %s (size: %d bytes)\n",
											fi.Name, formatType(fi.Field.Type), fi.TypeSize)
									} else {
										fmt.Printf("  %s (embedded, size: %d bytes)\n",
											formatType(fi.Field.Type), fi.TypeSize)
									}
								}

								newStruct := generateReorderedStruct(structType, reorderedFields)
								fmt.Println("\nReordered struct:")
								var buf strings.Builder
								format.Node(&buf, fset, &ast.GenDecl{
									Tok: token.TYPE,
									Specs: []ast.Spec{
										&ast.TypeSpec{
											Name: typeSpec.Name,
											Type: newStruct,
										},
									},
								})
								fmt.Println(buf.String())
							}
						}
					}
				}
			}
		}
		return true
	})

	if !hasIssues {
		fmt.Printf("✔ %s\n", filename)
		os.Exit(0)
	}

	return nil
}

func formatType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + formatType(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + formatType(t.Elt)
		}
		return "[...]" + formatType(t.Elt)
	case *ast.MapType:
		return "map[" + formatType(t.Key) + "]" + formatType(t.Value)
	case *ast.ChanType:
		return "chan " + formatType(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	}
	return "unknown"
}

func collectGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") && !strings.HasSuffix(d.Name(), "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func pwd() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("error finding current working directory:", err)
		return "", err
	}
	return dir, nil
}

func main() {
	var fix bool
	flag.BoolVar(&fix, "fix", false, "automatically fix struct field ordering issues")
	flag.Parse()

	targets := flag.Args()
	var files []string
	var err error

	cwd, _ := pwd()
	if len(targets) == 0 {
		// Default: process all .go files in current directory
		files, err = collectGoFiles(cwd)
		if err != nil {
			fmt.Printf("Failed to collect Go files: %v\n", err)
			os.Exit(1)
		}
	} else {
		info, err := os.Stat(targets[0])
		if err != nil {
			fmt.Printf("Invalid path: %v\n", err)
			os.Exit(1)
		}

		if info.IsDir() {
			files, err = collectGoFiles(targets[0])
			if err != nil {
				fmt.Printf("Failed to collect Go files from directory: %v\n", err)
				os.Exit(1)
			}
		} else {
			files = []string{targets[0]}
		}
	}

	hadIssues := false
	for _, file := range files {
		if fix {
			err = fixStructsInFile(file)
		} else {
			err = lintFile(file)
		}
		if err != nil {
			hadIssues = true
			fmt.Printf("Error in %s: %v\n", file, err)
		}
	}

	if hadIssues {
		os.Exit(1)
	}
}
