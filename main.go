package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/ast/astutil"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var err error

	src := flag.String("src", "", "")
	dst := flag.String("dst", "", "")
	modPath := flag.String("mod", "", "")
	vendor := flag.String("vendor", "zero-dep-vendor", "")

	flag.Parse()

	*src, err = filepath.Abs(*src)
	if err != nil {
		panic(err)
	}

	dstAbs, err := filepath.Abs(*dst)
	if err != nil {
		panic(err)
	}

	if *src == dstAbs {
		panic("src==dst")
	}

	excludeDirs := map[string]interface{}{
		".idea":   nil,
		".vscode": nil,
		".git":    nil,
	}

	oriModPath, mod, err := zeroDepMod(filepath.Join(*src, "go.mod"), *modPath)
	if err != nil {
		panic(err)
	}

	modFormatted, err := mod.Format()
	if err != nil {
		panic(err)
	}

	err = os.Mkdir(*dst, 0775)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filepath.Join(*dst, "go.mod"), modFormatted, 0664)
	if err != nil {
		panic(err)
	}

	err = filepath.Walk(*src, func(path string, info fs.FileInfo, _err error) error {
		rel, err := filepath.Rel(*src, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			_, ok := excludeDirs[rel]
			if ok {
				return filepath.SkipDir
			}
			return nil
		}

		dstPath := filepath.Join(*dst, rel)
		if strings.Index(rel, "vendor/") == 0 {
			dstPath = filepath.Join(*dst, *vendor, rel[len("vendor/"):])
		}

		switch filepath.Ext(info.Name()) {
		case ".mod", ".sum":
			{
				return nil
			}
		case ".go":
			{
				fileSet := token.NewFileSet()
				f, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
				if err != nil {
					return err
				}

				astutil.Apply(f, func(cr *astutil.Cursor) bool {
					switch cr.Node().(type) {
					case *ast.ImportSpec:
						{
							spec := cr.Node().(*ast.ImportSpec)
							if strings.Index(spec.Path.Value, ".") == -1 {
								return true
							}

							if strings.Index(spec.Path.Value, oriModPath) == 1 {
								spec.Path.Value = `"` + mod.Module.Mod.Path + spec.Path.Value[len(oriModPath)+1:]
							} else {
								spec.Path.Value = `"` + mod.Module.Mod.Path + "/" + *vendor + "/" +
									strings.ReplaceAll(spec.Path.Value, `"`, "") + `"`
							}

							cr.Replace(spec)
						}
					}
					return true
				}, nil)

				dstFile, err := createDst(dstPath)
				if err != nil {
					return err
				}
				defer dstFile.Close()

				printer.Fprint(dstFile, fileSet, f)
			}
		default:
			{
				dstFile, err := createDst(dstPath)
				if err != nil {
					return err
				}
				defer dstFile.Close()

				srcFile, err := os.Open(path)
				if err != nil {
					return err
				}
				defer srcFile.Close()

				_, err = io.Copy(dstFile, srcFile)
			}
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func createDst(dst string) (*os.File, error) {
	err := os.MkdirAll(filepath.Dir(dst), 0775)
	if err != nil {
		return nil, err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		return nil, err
	}
	return dstFile, nil
}

func zeroDepMod(p, modPath string) (oriModPath string, f2 *modfile.File, err error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}

	f, err := modfile.Parse(p, b, nil)
	if err != nil {
		return
	}

	oriModPath = f.Module.Mod.Path

	if modPath != "" {
		f.Module.Mod.Path = modPath
	}

	f2 = &modfile.File{
		Module: f.Module,
		Go:     f.Go,
		Syntax: &modfile.FileSyntax{},
	}

	for i := range f.Syntax.Stmt {
		stmt := f.Syntax.Stmt[i]
		switch stmt.(type) {
		case *modfile.Line:
			{
				if len(stmt.(*modfile.Line).Token) == 2 {
					switch stmt.(*modfile.Line).Token[0] {
					case "module":
						{
							stmt.(*modfile.Line).Token[1] = f.Module.Mod.Path
							f2.Syntax.Stmt = append(f2.Syntax.Stmt, stmt)
						}
					case "go":
						{
							f2.Syntax.Stmt = append(f2.Syntax.Stmt, stmt)
						}
					}
				}

			}
		case *modfile.CommentBlock:
			{
				f2.Syntax.Stmt = append(f2.Syntax.Stmt, stmt)
			}
		default:
			{
				//fmt.Println(reflect.TypeOf(stmt))
			}
		}
	}

	return
}
