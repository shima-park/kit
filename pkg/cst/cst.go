package cst

import (
	"bytes"
	"go/parser"
	"go/token"
)

type ConcreteSyntaxTree interface {
	PackageName() string
	Imports() []Import
	Vars() []Var
	Consts() []Constant
	Interfaces() []Interface
	Types() []Type
	Structs() []*Struct
	StructMap() map[string]map[string]*Struct
	Methods() []Method
	Parse() error
}

type Import struct {
	Alias string // 导入包的别名
	Path  string // 导入包的相对路径
}

type Interface struct {
	Name    string
	Methods []Method
}

type Method struct {
	Name    string
	Recv    []Field
	Params  []Field
	Results []Field
}

type Field struct {
	Pos  string
	Name string
	Type Type
	Tag  string
}

func New(filename string, opts ...Option) (ConcreteSyntaxTree, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}

	t := NewConcreteSyntaxTree(
		fset,
		f,
		opts...,
	)
	if err := t.Parse(); err != nil {
		return nil, err
	}
	return t, nil
}

func FieldsToString(fields []Field) string {
	buff := bytes.NewBufferString("")
	for i, field := range fields {
		buff.WriteString(field.Name + " " + field.Type.String())
		if i != len(fields)-1 {
			buff.WriteString(",")
		}
	}
	return buff.String()
}

func FieldsToStringFunc(separator string) func(fields []Field) string {
	return func(fields []Field) string {
		buff := bytes.NewBufferString("")
		for i, field := range fields {
			buff.WriteString(field.Name + " " + field.Type.String())
			if i != len(fields)-1 {
				buff.WriteString(separator)
			}
		}
		return buff.String()
	}
}

func FieldsKeyToStringFunc(separator string) func(fields []Field) string {
	return func(fields []Field) string {
		buff := bytes.NewBufferString("")
		for i, field := range fields {
			buff.WriteString(field.Name)
			if i != len(fields)-1 {
				buff.WriteString(separator)
			}
		}
		return buff.String()
	}
}

type GoType string

var (
	CrossProtocolUnsupportType GoType = "CrossProtocolUnsupportType"
	BasicType                  GoType = "BasicType"
	ArrayType                  GoType = "ArrayType"
	MapType                    GoType = "MapType"
	StructType                 GoType = "StructType"
	FuncType                   GoType = "FuncType"
	EllipsisType               GoType = "EllipsisType"
)

type Type struct {
	BaseType

	ElementType *BaseType

	KeyType   *BaseType
	ValueType *BaseType
}

type BaseType struct {
	Star     bool   // 指针类型
	X        string // 类型前缀或者说包名 例如: 括号包裹部分 (time.)Time  (*)XXXStruct
	Name     string // go显示类型名称 int, interface{}, XXXStruct
	GoType   GoType // basic, array, map, struct
	Position token.Position
}

func (t BaseType) String() string {
	switch t.GoType {
	case BasicType:
		buff := bytes.NewBufferString("")
		if t.Star {
			buff.WriteString("*")
		}

		if t.X != "" {
			buff.WriteString(t.X)
			buff.WriteString(".")
		}
		buff.WriteString(t.Name)
		return buff.String()
	case StructType:
		buff := bytes.NewBufferString("")
		if t.Star {
			buff.WriteString("*")
		}

		if t.X != "" {
			buff.WriteString(t.X)
			buff.WriteString(".")
		}
		buff.WriteString(t.Name)
		return buff.String()
	}
	return t.Name
}

func (t Type) String() string {
	switch t.GoType {
	case BasicType:
		buff := bytes.NewBufferString("")
		if t.Star {
			buff.WriteString("*")
		}

		if t.X != "" {
			buff.WriteString(t.X)
			buff.WriteString(".")
		}
		buff.WriteString(t.Name)
		return buff.String()
	case StructType:
		buff := bytes.NewBufferString("")
		if t.Star {
			buff.WriteString("*")
		}

		if t.X != "" {
			buff.WriteString(t.X)
			buff.WriteString(".")
		}
		buff.WriteString(t.Name)
		return buff.String()
	case ArrayType:
		buff := bytes.NewBufferString("[]")
		if t.Star {
			buff.WriteString("*")
		}

		if t.X != "" {
			buff.WriteString(t.X)
			buff.WriteString(".")
		}
		buff.WriteString(t.ElementType.String())
		return buff.String()
	case MapType:
		buff := bytes.NewBufferString("")
		if t.Star {
			buff.WriteString("*")
		}
		buff.WriteString("map[")
		buff.WriteString(t.KeyType.String())
		buff.WriteString("]")

		buff.WriteString(t.ValueType.String())
		return buff.String()
	}
	return t.Name
}

type Struct struct {
	Position    token.Position // 结构体位置
	PackageName string         // 所属的包名
	Name        string         // 结构体名称
	Fields      []Field        // 结构体字段
	Methods     []Method       // 结构体函数列表
	Type        *Type
}

type Var struct {
	Name  string
	Type  Type
	Value interface{}
}

type Constant struct {
	Name  string
	Type  Type
	Value interface{}
}
