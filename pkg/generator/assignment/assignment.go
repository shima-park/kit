package assignment

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
)

type GeneratorFactory struct {
	cst   cst.ConcreteSyntaxTree
	pbcst cst.ConcreteSyntaxTree
}

func NewGeneratorFactory(cst cst.ConcreteSyntaxTree, pbcst cst.ConcreteSyntaxTree) *GeneratorFactory {
	return &GeneratorFactory{cst: cst, pbcst: pbcst}
}

func findAssignmentStruct(dst *cst.Struct, srcs []gen.ReqAndResp) *cst.Struct {
	for _, src := range srcs {
		if dst.Name == src.Request.Name {
			return src.Request
		}

		if dst.Name == src.Response.Name {
			return src.Response
		}
	}
	return nil
}

func (g *GeneratorFactory) Generate(dst *cst.Struct, srcs []gen.ReqAndResp, srcAlias Alias) string {
	src := findAssignmentStruct(dst, srcs)
	if src != nil {
		buff := bytes.NewBufferString("")
		n := &AssignmentGenerator{
			pbcst:    g.pbcst,
			cst:      g.cst,
			dst:      dst,
			src:      src,
			writer:   buff,
			srcAlias: srcAlias,
		}
		err := n.Generate()
		if err != nil {
			panic(err)
		}

		return buff.String()
	}
	return ""
}

// 赋值方法生成器
type AssignmentGenerator struct {
	cst      cst.ConcreteSyntaxTree //
	pbcst    cst.ConcreteSyntaxTree
	writer   io.Writer
	src      *cst.Struct
	dst      *cst.Struct
	srcAlias Alias
}

func findStruct(packageName string, structName string, csts ...cst.ConcreteSyntaxTree) (*cst.Struct, bool) {
	var (
		s     *cst.Struct
		found bool
	)
	for _, cst := range csts {
		s, found = cst.StructMap()[packageName][structName]
		if found {
			return s, found
		}
	}
	return s, found
}

func (g *AssignmentGenerator) Generate() error {
	for _, srcField := range g.src.Fields {
		for _, dstField := range g.dst.Fields {
			if srcField.Name == dstField.Name {
				err := g.generateAssignmentSegment(g.srcAlias, srcField, dstField)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *AssignmentGenerator) findStruct(packageName string, structName string) *cst.Struct {
	switch packageName {
	case g.cst.PackageName():
		return g.cst.StructMap()[packageName][structName]
	case g.pbcst.PackageName():
		return g.pbcst.StructMap()[packageName][structName]
	}
	return nil
}

// 生成转换基本类型的方法体
func (g *AssignmentGenerator) generateBasicTypeAssignmentConvertFunc(srcAlias Alias, src cst.BaseType, dst cst.BaseType) {
	var (
		srcType   = src
		dstType   = dst
		aliasName = srcAlias.String()
	)
	if srcType.Name == dstType.Name {
		if srcType.Star && !dstType.Star {
			// F float64    req.F *float64
			// F: *req.F,
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (v %s) { if %s { v = *%s } ; return v }()",
					dstType, statement, aliasName)
			} else {
				g.print(" *%s ", aliasName)
			}

		} else if !srcType.Star && dstType.Star {
			// F *float64   req.F float64
			// F: &req.F,
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (v %s) { if %s { v = &%s } ; return v }()",
					dstType, statement, aliasName)
			} else {
				g.print(" &%s ", aliasName)
			}
		} else {
			// Message string    resp.Message string
			//Message: resp.Message,
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (v %s) { if %s { v = %s } ; return v }()",
					dstType, statement, aliasName)
			} else {
				g.print(" %s ", aliasName)
			}
		}
	} else {
		if srcType.Star && !dstType.Star {
			// T int64  req.T *int
			// T: int64(*req.T),
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (v %s) { if %s { v = %s(*%s) } ; return v }()",
					dstType, statement, dstType.Name, aliasName)
			} else {
				g.print(" %s(*%s) ", dstType.Name, aliasName)
			}
		} else if !srcType.Star && dstType.Star {
			// Y *int64    req.Y int
			// Y: func(i int) *int64 { return &i }(req.Y),
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (v %s) { if %s { k := %s(%s); v = &k } ; return v }()",
					dstType, statement, dstType.Name, aliasName)
			} else {
				g.print("func() (v %s) { k := %s(%s);v = &k ; return v }()",
					dstType, dstType.Name, aliasName)
			}
		} else {
			// Code int64  resp.Code int
			// Code: int64(resp.Code),
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (v %s) { if %s { v = %s(%s) } ; return v }()",
					dstType, statement, dstType.Name, aliasName)
			} else {
				g.print(" %s(%s) ", dstType.Name, aliasName)
			}
		}
	}
	return
}

// 生成结构体转换的方法体
func (g *AssignmentGenerator) generateStructTypeAssignmentConvertFunc(srcAlias Alias, src, dst cst.Field, srcStruct, dstStruct *cst.Struct) error {
	var (
		dstType = dst.Type.BaseType
		srcType = src.Type.BaseType
	)
	if dst.Type.ElementType != nil {
		// 当前父类型是数组时
		dstType = *dst.Type.ElementType
		srcType = *src.Type.ElementType
	} else if dst.Type.ValueType != nil {
		// 当前父类型是map时
		dstType = *dst.Type.ValueType
		srcType = *src.Type.ValueType
	}

	dstType.X = dstStruct.PackageName
	srcType.X = srcStruct.PackageName
	g.println("func(src %s) (dst %s) {", srcType.String(), dstType.String())
	{
		// 这里生成赋值方法的需要重新生成别名对象，需要从当前的src,dst信息中生成
		newAlias := NewObjectAlias(g.cst, g.pbcst)(
			"src",
			srcStruct.PackageName,
			srcStruct.Name,
			srcType.Star,
		)
		// 判断是否需要生成别名.字段名时的nil pointer检查语句
		statement, isNeed := newAlias.CheckNil()
		if isNeed {
			g.println("if %s {", statement)
		}

		{
			if dstType.Star {
				removeStarType := dstType
				removeStarType.Star = false
				g.println("dst = &%s{", removeStarType.String())
			} else {
				g.println("dst = %s{", dstType.String())
			}
			for _, srcField := range srcStruct.Fields {
				for _, dstField := range dstStruct.Fields {
					if srcField.Name == dstField.Name {
						err := g.generateAssignmentSegment(
							newAlias,
							srcField,
							dstField,
						)
						if err != nil {
							return err
						}
					}
				}
			}
			g.println("}")
		}

		if isNeed {
			g.println("}")
		}
	}
	g.print("; return dst}(%s)", srcAlias.String())
	return nil
}

func (g *AssignmentGenerator) generateAssignmentSegment(srcAlias Alias, src cst.Field, dst cst.Field) error {
	switch dst.Type.GoType {
	case cst.BasicType:
		g.print("%s: ", dst.Name)
		g.generateBasicTypeAssignmentConvertFunc(srcAlias.With(src.Name), src.Type.BaseType, dst.Type.BaseType)
		g.println(",")
	case cst.StructType:
		var (
			srcType = src.Type.BaseType
			dstType = dst.Type.BaseType
		)
		if dst.Type.ElementType != nil {
			srcType = *src.Type.ElementType
			dstType = *dst.Type.ElementType
		}
		// 寻找类型的数据结构
		srcStruct := g.findStruct(g.src.PackageName, srcType.Name)
		dstStruct := g.findStruct(g.dst.PackageName, dstType.Name)
		g.print("%s: ", dst.Name)
		err := g.generateStructTypeAssignmentConvertFunc(srcAlias.With(src.Name), src, dst, srcStruct, dstStruct)
		if err != nil {
			return err
		}
		g.println(",")
	case cst.ArrayType:
		switch dst.Type.ElementType.GoType {
		case cst.BasicType:
			// 基础类型相同的数组直接赋值，不需要转换
			// []string = []string, []int = []int ...
			if src.Type.Name == dst.Type.Name &&
				src.Type.Star == dst.Type.Star {
				g.println("%s: %s.%s,", dst.Name, srcAlias, src.Name)
				return nil
			}

			// 从父级别名新增引用
			srcAlias = srcAlias.With(src.Name)
			g.println("%s: func(src %s) (dst %s) {", dst.Name, src.Type.String(), dst.Type.String())
			{
				g.println("dst = make( %s, len(src))", dst.Type.String())
				g.println("for i, _ := range src{")
				{
					g.println("temp := src[i]")
					g.print("dst[i] = ")
					g.generateBasicTypeAssignmentConvertFunc(
						NewSimpleAlias("temp"),
						*src.Type.ElementType,
						*dst.Type.ElementType,
					)
					g.println("")
				}
				g.println("}")
				g.println("return")
			}
			g.println("}(%s),", srcAlias)
		case cst.StructType:
			// 数组的值是对象类型，生成转换方法
			srcStruct := g.findStruct(g.src.PackageName, src.Type.ElementType.Name)
			dstStruct := g.findStruct(g.dst.PackageName, dst.Type.ElementType.Name)

			src.Type.ElementType.X = srcStruct.PackageName
			dst.Type.ElementType.X = dstStruct.PackageName
			// 从父级别名新增引用
			srcAlias = srcAlias.With(src.Name)
			g.println("%s: func(src %s) (dst %s) {", dst.Name, src.Type.String(), dst.Type.String())
			{
				g.println("dst = make( %s, len(src))", dst.Type.String())
				g.println("for i, _ := range src{")
				{
					g.println("temp := src[i]")
					g.print("dst[i] = ")
					err := g.generateStructTypeAssignmentConvertFunc(
						srcAlias.ReplaceName("temp"),
						src,
						dst,
						srcStruct,
						dstStruct,
					)
					if err != nil {
						return err
					}
					g.println("")
				}
				g.println("}")
				g.println("return")
			}
			g.println("}(%s),", srcAlias)
		default:
			return errors.New("unsupport type" + dst.Type.String())
		}
	case cst.MapType:
		switch dst.Type.ValueType.GoType {
		case cst.BasicType:
			// 基础类型相同的map直接赋值，不需要转换
			// map[string]string = map[string]string
			if src.Type.Name == dst.Type.Name &&
				src.Type.Star == dst.Type.Star {
				g.println("%s: %s.%s,", dst.Name, srcAlias, src.Name)
				return nil
			}

			// 从父级别名新增引用
			srcAlias = srcAlias.With(src.Name)
			g.println("%s: func(src %s) (dst %s) {", dst.Name, src.Type.String(), dst.Type.String())
			{
				g.println("dst = make( %s, len(src))", dst.Type.String())
				g.println("for i, _ := range src{")
				{
					g.println("temp := src[i]")
					g.print("dst[")
					g.generateBasicTypeAssignmentConvertFunc(
						NewSimpleAlias("i"),
						*src.Type.KeyType,
						*dst.Type.KeyType,
					)
					g.print("] = ")
					g.generateBasicTypeAssignmentConvertFunc(
						NewSimpleAlias("temp"),
						*src.Type.ValueType,
						*dst.Type.ValueType,
					)
					g.println("")
				}
				g.println("}")
				g.println("return")
			}
			g.println("}(%s),", srcAlias)
		case cst.StructType:
			// map的值是对象类型，生成转换方法
			srcStruct := g.findStruct(g.src.PackageName, src.Type.ValueType.Name)
			dstStruct := g.findStruct(g.dst.PackageName, dst.Type.ValueType.Name)

			src.Type.ValueType.X = srcStruct.PackageName
			dst.Type.ValueType.X = dstStruct.PackageName
			// 从父级别名新增引用
			srcAlias = srcAlias.With(src.Name)
			g.println("%s: func(src %s) (dst %s) {", dst.Name, src.Type.String(), dst.Type.String())
			{
				g.println("dst = make(%s, len(src))", dst.Type.String())
				g.println("for k, v := range src{")
				{
					g.print("dst[k] =")
					err := g.generateStructTypeAssignmentConvertFunc(
						srcAlias.ReplaceName("v"),
						src,
						dst,
						srcStruct,
						dstStruct,
					)
					if err != nil {
						return err
					}
				}
				g.println("}")
				g.println("return")
			}
			g.println("}(%s),", srcAlias)
		default:
			return errors.New("unsupport type" + dst.Type.String())
		}
	}
	return nil
}

func (g *AssignmentGenerator) print(format string, a ...interface{}) {
	g.writer.Write([]byte(fmt.Sprintf(format, a...)))
}

func (g *AssignmentGenerator) println(format string, a ...interface{}) {
	g.writer.Write([]byte(fmt.Sprintf(format+"\n", a...)))
}
