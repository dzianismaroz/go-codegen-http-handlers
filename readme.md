Code generation is very widely used in Go and you need to know how to use this tool.

In this task, you will need to write a code generator that looks for structure methods marked with a special label and generates the following code for them:
* http wrappers for these methods
* authorization check
* method checks (GET/POST)
* parameter validation
* filling the structure with method parameters
* handling unknown errors

Those. you write a program (in the file `handlers_gen/codegen.go`) then run it, passing as parameters the path to the file for which you want to generate the code, and the path to the file in which to write the result. The run will look something like this: `go build handlers_gen/* && ./codegen api.go api_handlers.go`. Those. it will be launched as `binary_code generator what_parsim.go where_parsim.go`

No need to hardcode. All data - field names, available values, boundary values ​​- everything is taken from the structure itself, `struct tags apivalidator` and the code that we parse.

If you manually enter the name of the structure, which should end up in the resulting code after generation, then you are doing it wrong, even if your tests pass. Your code generator should work universally for any fields and values ​​that it knows. You need to write code so that it works on code that is unknown to you, similar to api.go.

The only thing you can use is `type ApiError struct` when checking an error. We believe that this is some kind of well-known structure.

The code generator can process the following types of structure fields:
* `int`
* `string`

The following placeholder validator labels `apivalidator` are available to us:
* `required` - the field must not be empty (should not have a default value)
* `paramname` - if specified, then take from the parameter with this name, otherwise `lowercase` from the name
* `enum` - "one of"
* `default` - if specified and an empty value is received (default value) - set what is written specified in `default`
* `min` - >= X for type `int`, for strings `len(str)` >=
* `max` - <= X for type `int`

See the tests for the error format. Error order:
* presence of a method (in `ServeHTTP`)
* method (POST)
* authorization
* parameters in the order they appear in the structure

Authorization is checked simply by the fact that the value `100500` was received in the header

The generated code will have something like this:

`ServeHTTP` - accepts all methods from the multiplexer, if found - calls `handler$methodName`, if not - says `404`
`handler$methodName` - wrapper over the method of the `$methodName` structure - performs all checks, displays errors or results in `JSON` format
`$methodName` is directly the method of the structure for which we generate the code and which we parse. Prefixed with `apigen:api` followed by `json` with the method name, type and authorization requirement. There is no need to generate it, it already exists.

```go
type SomeStructName struct{}

func (h *SomeStructName ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
 switch r.URL.Path {
 case "...":
 h.wrapperDoSomeJob(w, r)
 default:
 // 404
 }
}

func (h *SomeStructName ) wrapperDoSomeJob() {
 // filling the params structure
 // parameter validation
 res, err := h.DoSomeJob(ctx, params)
 // other processing
}
```

According to the structure of the code generator - you need to find all the methods, for each method generate validation of incoming parameters and other checks in `handler$methodName`, for a pack of structure methods generate binding in `ServeHTTP`

You don’t have to worry too much about the errors of the code generator - the parameters that are passed to it will be considered guaranteed to be correct.

What needs to be parsed in ast:
* `node.Decls` -> `ast.FuncDecl` is a method. it needs to check that it has a label and start generating a wrapper for it
* `node.Decls` -> `ast.GenDecl` -> `spec.(*ast.TypeSpec)` + `currType.Type.(*ast.StructType)` is a structure. it is needed to generate validation for the method that we found in the previous paragraph
* https://golang.org/pkg/go/ast/#FuncDecl - here see what structure the method belongs to

Adviсe:
* You can use either templates to generate the entire method at once, or assemble the code from small pieces.
* The easiest way to implement it is in 2 passes - in the first one, collect everything that needs to be generated, in the second - the actual code generation
* You will need to convert a lot from interfaces, just look at what is there `fmt.Printf("type: %T data: %+v\n", val, val)`

Directory structure:
* example/ - example with code generation from the 3rd lecture of the 1st part of the course. You can use this code as a basis.
* handlers_gen/codegen.go - this is where you write code
* api.go - you need to feed this file to the code generator. no need to edit it
* main.go - everything is clear here. no need to edit
* main_test.go - this file should be run for testing after code generation. no need to edit

The tests will run like this:
```shell
# while in this folder
# .exe extension is only for lucky Windows owners
# collecting
