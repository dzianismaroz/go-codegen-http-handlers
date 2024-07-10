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
	// Template paths.
	validatorTplPath = "./handlers_gen/templates/api_validator.tmpl"
	handlerTplPath   = "./handlers_gen/templates/func_handler.tmpl"
	// Codegen annotations.
	apiGenAnnotation       = "apigen:api"
	apiValidatorAnnotation = "apivalidator:"
	// Paramname validators
	paramNameValidator    = "paramname"
	defaultValueValidator = "default"
	minValueValidator     = "min"
	maxValueValidator     = "max"
	enumValuesValidator   = "enum"
	// Types of validated struct fields.
	StringFieldType FieldType = false
	IntFieldType    FieldType = true
)

// Load text template file for code generation. On any problem - panic and prevent execution.
func loadTemplate(tplPath string) *template.Template {
	if templ, err := template.ParseFiles(tplPath); err != nil {
		panic(fmt.Sprintf("failed to parse template [%s]: %s", tplPath, err.Error()))
	} else {
		return templ
	}
}

// Parse AST node : it must be of type Function and contain comments with codegen marker: '// apigen:api'.
func isFuncCodegen(a ast.Decl) (*ApiGen, bool) {
	if f, isFunc := a.(*ast.FuncDecl); !isFunc || f.Doc == nil {
		return nil, false // If not a function - we are not intrested
	} else {
		for _, comment := range f.Doc.List {
			if strings.Contains(comment.Text, apiGenAnnotation) { // If functions has no annotation - skip it.
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
					continue // If not a struct - we are not intrested
				} else {
					var result = &StructValidator{
						StructName: currType.Name.Name,
						Validators: make(map[FieldName]*FieldValidator, len(currStruct.Fields.List))}
					var gotResult bool
					// search any field contains codegen marker. Otherwise - will be ignored.
					for _, field := range currStruct.Fields.List {
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

// Parse annotated struct field and return aggregated information of it.
func produceValidator(f *ast.Field) (result *FieldValidator) {
	tag := f.Tag.Value
	if len(tag) < 10 {
		return nil
	}
	// Purify string content for tokenizing annotation values.
	// i.e:  apivalidator:"enum=user|moderator|admin,default=user" -> enum=user|moderator|admin,default=user
	tag = strings.ReplaceAll(tag, apiValidatorAnnotation, "")
	tag = strings.ReplaceAll(tag, "\"", "")
	tag = strings.ReplaceAll(tag, "`", "")
	validators := strings.Split(tag, ",")
	// Prepare result
	result = &FieldValidator{FieldType: f.Type.(*ast.Ident).Name == "int", ParamName: strings.ToLower(f.Names[0].Name)}

	for i := 0; i < len(validators); i++ {
		v := validators[i]
		if !strings.Contains(v, "=") && v == "required" {
			result.Required = true
			continue
		}
		kv := strings.Split(v, "=") // Expecting annotation value presented. Otherwise panicked.
		switch kv[0] {
		case paramNameValidator:
			result.ParamName = kv[1]
		case defaultValueValidator:
			result.Default = kv[1]
		case minValueValidator:
			if intVal, err := strconv.Atoi(kv[1]); err != nil {
				panic("invalid int value for validator `min`:" + kv[1])
			} else {
				result.Min = intVal
			}
		case maxValueValidator:
			if intVal, err := strconv.Atoi(kv[1]); err != nil {
				panic("invalid int value for validator `max`:" + kv[1])
			} else {
				result.Max = intVal
			}
		case enumValuesValidator:
			result.Enum = strings.Split(kv[1], "|")
		default:
			panic("unknown validator : " + kv[0]) // Prevent futher exection - wrong annotation detected.
		}
	}
	return
}

func parseSourceFile() (funcsForCodegen map[StructReceiver]Methods, structsForCodegen []*StructValidator) {
	// Collect metadata of target functions and structs to be used for code generating.
	funcsForCodegen = make(map[StructReceiver]Methods, 50)
	structsForCodegen = make([]*StructValidator, 0, 50)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)

	if err != nil {
		log.Fatal(err)
	}

	var structReceiver StructReceiver
	for _, f := range node.Decls {
		if fn, ok := isFuncCodegen(f); ok { // Parse source code file for annotated functions and collect info.
			structReceiver = fn.receiver // use as key a name of struct - receiver of the method
			funcsForCodegen[structReceiver] = append(funcsForCodegen[structReceiver], fn)
		}
		if st, ok := isStructCodegen(f); ok { // Parse source code file for annotated structs and collect info.
			structsForCodegen = append(structsForCodegen, st)
		}
	}
	return
}

// Extract apigen annotation for detected struct.
// i.e. apigen:api {"url": "/user/create", "auth": true, "method": "POST"} -> {Url: /user/create, Auth: false, Method: POST}
func exctractApiGenAnnotation(f *ast.FuncDecl) (api *ApiGen, err error) {
	api = &ApiGen{}
	// Purify string by removing `apigen:api` and return result as ApiGen struct.
	err = json.Unmarshal([]byte(strings.ReplaceAll(f.Doc.Text(), apiGenAnnotation, "")), api)
	return
}

// Genereate required code wrappers for detected funcions.
func handleFuncsCodegen(funcs map[StructReceiver][]*ApiGen, out *os.File, templ *template.Template) {
	if err := templ.Execute(out, funcs); err != nil {
		panic(fmt.Errorf("failed to process template [%s]: %s", templ.Name(), err.Error()))
	}
}

// Genereate required validation code wrappers for detected structs.
func handleStructsCodegen(structs []*StructValidator, out *os.File, templ *template.Template) {
	if err := templ.Execute(out, structs); err != nil {
		panic(fmt.Errorf("failed to process template [%s]: %s", templ.Name(), err.Error()))
	}
}

func main() {
	// Load templates content. Fail with error immediately on any problem.
	handlerTemplate := loadTemplate(handlerTplPath)
	validatorTemplate := loadTemplate(validatorTplPath)
	// Parse source code file which we need to generate wrappers.
	funcsForCodegen, structsForCodegen := parseSourceFile()
	out, _ := os.Create(os.Args[2])
	defer out.Close()

	handleFuncsCodegen(funcsForCodegen, out, handlerTemplate)       // Generate http-handlers wrappers for functions.
	handleStructsCodegen(structsForCodegen, out, validatorTemplate) // Generate validators wrappers for structs.
}
