package main

import (
	"go/ast"
	"strings"
)

type (
	StructReceiver = string    // name of struct- receiver of method to be codegen ( used as key in map)
	FieldName      = string    // name of field to apply validator
	Methods        = []*ApiGen // methods resolved to be used for coedegen
	FieldType      = bool      // type of struct field to apply validation and pass value from request
)

// Struct to aggregate infromation about method found for codegen appliance
type ApiGen struct {
	Url      string        `json:"url"`
	Auth     bool          `json:"auth"`
	Method   string        `json:"method"`
	Target   *ast.FuncDecl // target function for http-wrapper codegen
	receiver string        // name to struct as method receiver ( used as key in map)
	ArgType  ast.Expr
}

// Aggregate information on every struct field to apply validation
type FieldValidator struct {
	ParamName string    // name of request param . default - field name on lowercase
	FieldType FieldType // string  \ int
	Required  bool
	Enum      []string // allowed predefined values
	Default   interface{}
	Min, Max  int // value of int. or lenght of string
}

func (fv *FieldValidator) HasEnumConstraint() bool {
	return len(fv.Enum) > 0
}

func (fv *FieldValidator) HasMinConstraint() bool {
	return fv.Min > 0
}

func (fv *FieldValidator) HasMaxConstraint() bool {
	return fv.Max > 0
}

func (fv *FieldValidator) StringifyEnum() string {
	return strings.Join(fv.Enum, " ")
}

type StructValidator struct {
	StructName string
	Validators map[FieldName]*FieldValidator
}
