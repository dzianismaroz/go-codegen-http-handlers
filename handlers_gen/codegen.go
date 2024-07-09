package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
)

const (
	// template paths
	validatorTplPath = "./handlers_gen/templates/api_validator.tmpl"
	handlerTplPath   = "./handlers_gen/templates/func_handler.tmpl"
	// codegen annotations
	apiGenAnnotation       = "apigen:api"
	apiValidatorAnnotation = "apivalidator:"
	// types of validated struct fields
	StringFieldType FieldType = false
	IntFieldType    FieldType = true
)

func loadTemplate(tplPath string) *template.Template {
	if templ, err := template.ParseFiles(tplPath); err != nil {
		panic(fmt.Sprintf("failed to parse templates: %s", tplPath))
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
				if api, err := exctractApiGenAnnotation(f); err != nil {
					panic(fmt.Sprintf("failed to parse annotation on %s", comment.Text))
				} else {
					api.Target = f
					api.ArgType = f.Type.Params.List[1].Type
					api.receiver = f.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name // name of struct-receiver
					return api, true
				}
			}
		}
		return nil, false
	}
}

// Parse AST node : it must be of type struct and contain any field with codegen marker: 'apivalidator'.
func isStructCodegen(a ast.Decl) (*StructValidator, bool) {
	if f, isGen := a.(*ast.GenDecl); !isGen { // `node.Decls` -> `ast.GenDecl`
		return nil, false
	} else { //
		for _, spec := range f.Specs {
			if currType, ok := spec.(*ast.TypeSpec); !ok { // `ast.GenDecl` -> `spec.(*ast.TypeSpec)`
				continue
			} else {
				if currStruct, ok := currType.Type.(*ast.StructType); !ok { // `spec.(*ast.TypeSpec)` ->  `currType.Type.(*ast.StructType)`
					continue
				} else {
					// res :=
					var result = &StructValidator{
						StructName: currType.Name.Name,
						Validators: make(map[FieldName]*FieldValidator, len(currStruct.Fields.List))}
					var gotResult bool
					for _, field := range currStruct.Fields.List { // search any field contains codegen marker
						if field.Tag != nil && strings.Contains(field.Tag.Value, apiValidatorAnnotation) {
							result.Validators[field.Names[0].Name] = produceValidator(field)
							gotResult = true
						}
					}
					return result, gotResult
				}
			}
		}
		return nil, false
	}
}

func produceValidator(f *ast.Field) (result *FieldValidator) {
	tag := f.Tag.Value
	if len(tag) < 10 {
		return nil
	}
	tag = strings.ReplaceAll(tag, apiValidatorAnnotation, "")
	tag = strings.ReplaceAll(tag, "\"", "")
	tag = strings.ReplaceAll(tag, "`", "")
	validators := strings.Split(tag, ",")
	result = &FieldValidator{FieldType: f.Type.(*ast.Ident).Name == "int", ParamName: strings.ToLower(f.Names[0].Name)}

	for i := 0; i < len(validators); i++ {
		v := validators[i]
		if !strings.Contains(v, "=") && v == "required" {
			result.Required = true
			continue
		}
		kv := strings.Split(v, "=")
		switch kv[0] {
		case "paramname":
			result.ParamName = kv[1]
		case "default":
			result.Default = kv[1]
		case "min":
			if intVal, err := strconv.Atoi(kv[1]); err != nil {
				panic("invalid int value for validator `min`:" + kv[1])
			} else {
				result.Min = intVal
			}
		case "max":
			if intVal, err := strconv.Atoi(kv[1]); err != nil {
				panic("invalid int value for validator `max`:" + kv[1])
			} else {
				result.Max = intVal
			}
		case "enum":
			result.Enum = strings.Split(kv[1], "|")
		default:
			panic("unknown validator : " + kv[0])
		}
	}
	return
}

func parseSourceFile() (funcsForCodegen map[StructReceiver]Methods, structsForCodegen []*StructValidator) {
	funcsForCodegen = make(map[StructReceiver]Methods, 10)
	structsForCodegen = make([]*StructValidator, 0, 50)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)

	if err != nil {
		log.Fatal(err)
	}

	var structReceiver StructReceiver
	for _, f := range node.Decls {
		if fn, ok := isFuncCodegen(f); ok {
			structReceiver = fn.receiver // use as key a name of struct - receiver of the method
			funcsForCodegen[structReceiver] = append(funcsForCodegen[structReceiver], fn)
		}
		if st, ok := isStructCodegen(f); ok {
			structsForCodegen = append(structsForCodegen, st)
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
	if err := templ.Execute(out, funcs); err != nil {
		panic(fmt.Errorf("failed to process template: %s", err.Error()))
	}
}

// Genereate required code for structs validation.
func handleStructsCodegen(structs []*StructValidator, out *os.File, templ *template.Template) {
	if err := templ.Execute(out, structs); err != nil {
		panic(fmt.Errorf("failed to process template: %s", err.Error()))
	}
}

func main() {
	handlerTemplate := loadTemplate(handlerTplPath)
	validatorTemplate := loadTemplate(validatorTplPath)
	funcsForCodegen, structsForCodegen := parseSourceFile()
	out, _ := os.Create(os.Args[2])
	defer out.Close()

	handleFuncsCodegen(funcsForCodegen, out, handlerTemplate)
	handleStructsCodegen(structsForCodegen, out, validatorTemplate)
}
