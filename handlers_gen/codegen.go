package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"log"
	"os"
	"strings"
)

const (
	// template paths
	validatorTplPath = "./handlers_gen/templates/api_validator.tpl"
	handlerTplPath   = "./handlers_gen/templates/func_handler.tpl"
	// codegen annotations
	apiGenAnnotation       = "apigen:api"
	apiValidatorAnnotation = "apivalidator"
	// header of target generated file
	header = `//generated content. do not edit
package main

import "net/http"
`
)

type StructReceiver = string
type Methods = []*ApiGen

type ApiGen struct {
	Url      string `json:"url"`
	Auth     bool   `json:"auth"`
	Method   string `json:"method"`
	target   *ast.FuncDecl
	receiver string
}

func loadTemplate(tplPath string) *template.Template {
	if templ, err := template.ParseFiles(tplPath); err != nil {
		panic("failed to parse templates:" + tplPath)
	} else {
		return templ
	}
}

// Parse AST node : it must be of type Function and contain comments with codegen marker: '// apigen:api'.
func isFuncCodegen(a ast.Decl) (*ApiGen, bool) {
	if f, isFunc := a.(*ast.FuncDecl); !isFunc || f.Doc == nil {
		return nil, false
	} else {
		for _, comment := range f.Doc.List {
			if strings.Contains(comment.Text, apiGenAnnotation) {
				api, err := exctractApiGenAnnotation(f)
				if err != nil {
					panic("failed to parse annotation on" + comment.Text)

				}
				api.target = f
				api.receiver = f.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
				return api, true
			}
		}
		return nil, false
	}
}

// Parse AST node : it must be of type struct and contain any field with codegen marker: 'apivalidator'.
func isStructCodegen(a ast.Decl) bool {
	if f, isGen := a.(*ast.GenDecl); !isGen { // `node.Decls` -> `ast.GenDecl`
		return false
	} else { //
		for _, spec := range f.Specs {
			if currType, ok := spec.(*ast.TypeSpec); !ok { // `ast.GenDecl` -> `spec.(*ast.TypeSpec)`
				continue
			} else {
				if currStruct, ok := currType.Type.(*ast.StructType); !ok { // `spec.(*ast.TypeSpec)` ->  `currType.Type.(*ast.StructType)`
					continue
				} else {
					for _, field := range currStruct.Fields.List { // search any field contains codegen marker
						if field.Tag != nil && strings.Contains(field.Tag.Value, apiValidatorAnnotation) {
							return true
						}
					}
				}
			}
		}
		return false
	}
}

func parseSourceFile() (funcsForCodegen map[StructReceiver]Methods, structsForCodegen []ast.Decl) {
	funcsForCodegen = make(map[StructReceiver]Methods, 10)
	structsForCodegen = make([]ast.Decl, 0, 50)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)

	if err != nil {
		log.Fatal(err)
	}

	for _, f := range node.Decls {
		if fn, ok := isFuncCodegen(f); ok {
			funcsForCodegen[fn.receiver] = append(funcsForCodegen[fn.receiver], fn)
		}
		if isStructCodegen(f) {
			structsForCodegen = append(structsForCodegen, f)
		}
	}
	return
}

// Extract apigen annotation as struct.
// i.e. apigen:api {"url": "/user/create", "auth": true, "method": "POST"} -> {Url: /user/create, Auth: false, Method: POST}
func exctractApiGenAnnotation(f *ast.FuncDecl) (api *ApiGen, err error) {
	api = &ApiGen{}
	err = json.Unmarshal([]byte(strings.ReplaceAll(f.Doc.Text(), apiGenAnnotation, "")), api)
	return
}

// Genereate required code for funcions.
func handleFuncsCodegen(funcs map[StructReceiver][]*ApiGen, out *os.File, templ *template.Template) {
	// fns := make(map[string])
	for receiver, apis := range funcs {
		if err := templ.Execute(out, api); err != nil {
			panic("failed to process template: " + err.Error())
		}
	}
}

// Genereate required code for structs validation.
func handleStructsCodegen(funcs []ast.Decl, out *os.File, templ *template.Template) {
}

func main() {
	handlerTemplate := loadTemplate(handlerTplPath)
	validatorTemplate := loadTemplate(validatorTplPath) // load template for code generate from source file
	funcsForCodegen, structsForCodegen := parseSourceFile()
	out, _ := os.Create(os.Args[2])
	defer out.Close()

	if _, err := out.WriteString(header); err != nil {
		panic(" error while writing target content: " + err.Error())
	}

	handleFuncsCodegen(funcsForCodegen, out, handlerTemplate)
	handleStructsCodegen(structsForCodegen, out, validatorTemplate)

}
