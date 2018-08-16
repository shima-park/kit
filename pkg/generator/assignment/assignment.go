package assignment

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
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
	// structmap中会包含其他引用包中的strut
	// 不能简单粗暴根据packageName进行switch
	if s, found := g.cst.StructMap()[packageName][structName]; found {
		return s
	}

	if s, found := g.pbcst.StructMap()[packageName][structName]; found {
		return s
	}
	return nil
}

// 生成转换基本类型的方法体
func (g *AssignmentGenerator) generateBasicTypeAssignmentConvertFunc(srcAlias Alias, srcType cst.BaseType, dstType cst.BaseType) {
	var aliasName = srcAlias.String()
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
func (g *AssignmentGenerator) generateStructTypeAssignmentConvertFunc(srcAlias Alias, srcType, dstType cst.BaseType, srcStruct, dstStruct *cst.Struct) error {
	// 非当前包去生成赋值语句时，需要补全引用的包名
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
		statement, isNeedWrap := newAlias.CheckNil()
		if isNeedWrap {
			g.println("if %s {", statement)
		}

		{
			if dstStruct.Type != nil && srcStruct.Type != nil {
				// 枚举类型互相转换
				srcStruct.Type.BaseType.Star = srcType.Star
				if dstType.Star {
					removeStarType := dstType
					removeStarType.Star = false
					g.print("temp := %s(", removeStarType.String())
					g.generateBasicTypeAssignmentConvertFunc(newAlias, srcStruct.Type.BaseType, dstStruct.Type.BaseType)
					g.println(")")
					g.print("dst = &temp")
				} else {
					g.print("dst = %s(", dstType)
					g.generateBasicTypeAssignmentConvertFunc(newAlias, srcStruct.Type.BaseType, dstStruct.Type.BaseType)
					g.print(")")
				}
			} else {
				if dstType.Star {
					removeStarType := dstType
					removeStarType.Star = false
					g.println("dst = &%s{", removeStarType.String())
				} else {
					g.println("dst = %s{", dstType)
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
		}

		if isNeedWrap {
			g.println("}")
		}
	}
	g.print("; return dst}(%s)", srcAlias.String())
	return nil
}

// TODO 如果有同样包名的就会有问题
// 0 model {./pkg/addservice/service.go:47:2 C model.Misc2 }, 数据类型X标明了引用包名
// 1 model {/Users/liuxingwang/go/src/ezrpro.com/micro/demo/model/misc.go:9:2 C Foo }, 没有表明引用包名，尝试从文件定义处获取包名
// 2 直接使用当前语法树的packageName
func inferPackageName(t cst.BaseType, def string) string {
	return ifEmpty(
		t.X,
		utils.GetPackageNameByFileAbsPath(t.Position.String()),
		def,
	)
}

func ifEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func (g *AssignmentGenerator) generateAssignmentSegment(srcAlias Alias, src cst.Field, dst cst.Field) error {
	switch dst.Type.GoType {
	case cst.BasicType:
		g.print("%s: ", dst.Name)
		g.generateBasicTypeAssignmentConvertFunc(srcAlias.With(src.Name), src.Type.BaseType, dst.Type.BaseType)
		g.println(",")
	case cst.StructType:
		srcType := src.Type.BaseType
		dstType := dst.Type.BaseType
		// 寻找类型的数据结构
		srcStruct := g.findStruct(inferPackageName(srcType, g.src.PackageName), srcType.Name)
		dstStruct := g.findStruct(inferPackageName(dstType, g.dst.PackageName), dstType.Name)
		g.print("%s: ", dst.Name)
		err := g.generateStructTypeAssignmentConvertFunc(srcAlias.With(src.Name), srcType, dstType, srcStruct, dstStruct)
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
			srcStruct := g.findStruct(inferPackageName(*src.Type.ElementType, g.src.PackageName), src.Type.ElementType.Name)
			dstStruct := g.findStruct(inferPackageName(*dst.Type.ElementType, g.dst.PackageName), dst.Type.ElementType.Name)

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
						*src.Type.ElementType,
						*dst.Type.ElementType,
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
			srcStruct := g.findStruct(inferPackageName(*src.Type.ValueType, g.src.PackageName), src.Type.ValueType.Name)
			dstStruct := g.findStruct(inferPackageName(*dst.Type.ValueType, g.dst.PackageName), dst.Type.ValueType.Name)

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
						*src.Type.ValueType,
						*dst.Type.ValueType,
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
