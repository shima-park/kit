package transport

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
)

type TransportGenerator struct {
	cst  cst.ConcreteSyntaxTree
	opts Options
}

func NewTransportGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
	options := newOptions(opts...)

	return &TransportGenerator{
		cst:  t,
		opts: options,
	}
}

func (g *TransportGenerator) Generate() error {
	for tplName, readWriter := range g.opts.readWriterMap {
		tplBody, err := ioutil.ReadAll(readWriter.template)
		if err != nil {
			return err
		}

		pbCST, err := getProtobufCST(g.opts.baseServiceName, g.cst.PackageName())
		if err != nil {
			return err
		}

		t := template.New(string(tplName)).Funcs(map[string]interface{}{
			"ToLowerFirstCamelCase":     utils.ToLowerFirstCamelCase,
			"ToCamelCase":               utils.ToCamelCase,
			"BasePath":                  filepath.Base,
			"ToLower":                   strings.ToLower,
			"GenerateAssignmentSegment": NewAssignmentGeneratorFactory(g.cst, pbCST).Generate,
			"NewSimpleAlias":            NewSimpleAlias,
			"NewObjectAlias":            NewObjectAlias(g.cst, pbCST),
		})
		t, err = t.Parse(string(tplBody))
		if err != nil {
			return err
		}

		serviceIface, err := gen.FilterInterface(g.cst.Interfaces(), g.opts.serviceSuffix)
		if err != nil {
			return err
		}

		// protobuf的interface是以server结尾
		pbServiceIface, err := gen.FilterInterface(pbCST.Interfaces(), utils.GetProtobufServiceSuffix())
		if err != nil {
			return err
		}

		err = t.Execute(readWriter.writer, map[string]interface{}{
			"BaseServiceName":        g.opts.baseServiceName,
			"PackageName":            g.opts.transportPackageName,
			"ServiceName":            serviceIface.Name,
			"ServiceMethods":         serviceIface.Methods,
			"ServiceImportPath":      utils.GetServiceImportPath(g.opts.baseServiceName),
			"EndpointImportPath":     utils.GetEndpointImportPath(g.opts.baseServiceName),
			"ProtobufImportPath":     utils.GetProtobufImportPath(g.opts.baseServiceName),
			"RequestAndResponseList": gen.GetRequestAndResponseList(g.cst),
			"ProtobufCST": map[string]interface{}{
				"PackageName":            pbCST.PackageName(),
				"ServiceName":            pbServiceIface.Name,
				"RequestAndResponseList": gen.GetRequestAndResponseList(pbCST),
			},
		})
	}
	return nil
}

func getProtobufCST(baseServiceName, servicePackageName string) (cst.ConcreteSyntaxTree, error) {
	pbGoPath := utils.GetProtobufFilePath(baseServiceName)

	pbGoFile := filepath.Join(pbGoPath, servicePackageName+".pb.go")
	pbCST, err := cst.New(pbGoFile)
	if err != nil {
		return nil, err
	}
	return pbCST, nil
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

type AssignmentGeneratorFactory struct {
	cst   cst.ConcreteSyntaxTree
	pbcst cst.ConcreteSyntaxTree
}

func NewAssignmentGeneratorFactory(cst cst.ConcreteSyntaxTree, pbcst cst.ConcreteSyntaxTree) *AssignmentGeneratorFactory {
	return &AssignmentGeneratorFactory{cst: cst, pbcst: pbcst}
}

func (g *AssignmentGeneratorFactory) Generate(dst *cst.Struct, srcs []gen.ReqAndResp, srcAlias Alias) string {
	src := findAssignmentStruct(dst, srcs)
	if src != nil {
		n := &AssignmentGenerator{
			pbcst:  g.pbcst,
			cst:    g.cst,
			dst:    dst,
			src:    src,
			writer: bytes.NewBufferString(""),
		}
		return n.Generate(srcAlias)
	}
	return ""
}

// 赋值方法生成器
type AssignmentGenerator struct {
	cst           cst.ConcreteSyntaxTree
	pbcst         cst.ConcreteSyntaxTree
	writer        *bytes.Buffer
	src           *cst.Struct
	dst           *cst.Struct
	referenceType map[string]struct{} // key: [struct.Name or type.Name] val: struct{}{}
}

type Alias interface {
	CheckNil() (statement string, isNeed bool)
	String() string
	IsStar() bool
	With(sub string) Alias
	ReplaceName(name string) Alias
	ReplaceRootStruct(s *cst.Struct) Alias
}

// 简单别名实现，这个目前用在map或者slice时,赋值的方法创建的临时变量
type SimpleAlias struct {
	name string
}

func NewSimpleAlias(name string) Alias {
	return SimpleAlias{name: name}
}

func (sa SimpleAlias) CheckNil() (statement string, isNeed bool) {
	return "", false
}

func (sa SimpleAlias) String() string {
	return sa.name
}

func (sa SimpleAlias) IsStar() bool {
	return false
}

func (sa SimpleAlias) With(sub string) Alias {
	sa.name += "." + sub
	return sa
}

func (sa SimpleAlias) ReplaceName(name string) Alias {
	sa.name = name
	return sa
}

func (sa SimpleAlias) ReplaceRootStruct(s *cst.Struct) Alias {
	return sa
}

// 对象别名实现，创建时传入根别名，是否是指针类型，及别名的对象数据结构
// 往下传递时，通过With增加 例如当alias.name=resp 向下引用 resp.Data = alias.With("Data")
// 生成检查空指针方法时，通过遍历rootStruct字段和分割的alias对应的字段，
// 通过csts数组查找Struct并获取其类型，判断是否需要生成检查空指针方法
type ObjectAlias struct {
	name       string
	csts       []cst.ConcreteSyntaxTree
	rootStruct *cst.Struct
	isStar     bool
}

func NewObjectAlias(csts ...cst.ConcreteSyntaxTree) func(name string, pkg, structName string, isStar bool) Alias {
	return func(name, pkg, structName string, isStar bool) Alias {
		rootStruct, found := findStruct(pkg, structName, csts...)
		if !found {
			panic(fmt.Sprintf("Not found struct(%s, %s) ", pkg, structName))
		}
		return ObjectAlias{
			name:       name,
			rootStruct: rootStruct,
			isStar:     isStar,
			csts:       csts,
		}
	}
}

// 生成检查别名引用是否为空指针的方法申明
// resp.Data.Name
// if resp != nil && resp.Data != nil {...}
func (oa ObjectAlias) CheckNil() (statement string, isNeed bool) {
	aliases := strings.Split(oa.name, ".")
	concat := []string{}
	conditions := []string{}
	pkg := oa.rootStruct.PackageName
	tempStruct := oa.rootStruct
	for i, alias := range aliases {
		// concat 需要拼接当前遍历的引用项
		// 例如A.B.C.D
		// concat: 第一次1. A 第二次2. A.B
		// conditions: 1. A != nil  2. A != nil && A.B != nil
		concat = append(concat, alias)
		// 通过.切割获得的resp，无法知道其类型，只能从创建alias时就获取它的信息
		// 例如resp.Data.Name中的resp,在创建时获知其是不是一个指针类型
		if i == 0 {
			if oa.isStar {
				conditions = append(conditions, fmt.Sprintf(" %s != nil ", strings.Join(concat, ".")))
			}
		} else {
			for _, field := range tempStruct.Fields {
				// 获取当前匹配的别名使用字段,非当前别名引用字段跳过
				if field.Name != alias {
					continue
				}
				// 获取到深层次的数据类型
				fieldType := field.Type.BaseType
				if field.Type.ElementType != nil {
					fieldType = *field.Type.ElementType
				} else if field.Type.ValueType != nil {
					fieldType = *field.Type.ValueType
				}

				// 如果时指针类型，拼接到条件列表里
				if field.Type.Star {
					conditions = append(conditions, fmt.Sprintf(" %s != nil ", strings.Join(concat, ".")))
				}

				// 引用到基础类型的指针*int,*string,*float...不需要再向下找引用的数据结构
				if fieldType.GoType == cst.BasicType {
					continue
				} else {
					// 通过packagename和字段的类型名从关联的strcutmap中查找对应的数据结构
					typeStruct, found := findStruct(pkg, fieldType.Name, oa.csts...)
					if found {
						tempStruct = typeStruct
					} else {
						panic(fmt.Sprintf("not found struct(%s)", fieldType.String()))
					}
				}
			}
		}
	}

	if len(conditions) > 0 {
		return strings.Join(conditions, "&&"), true
	}

	return "", false
}

func (oa ObjectAlias) String() string {
	return oa.name
}

func (oa ObjectAlias) IsStar() bool {
	return oa.isStar
}

func (oa ObjectAlias) With(sub string) Alias {
	oa.name += "." + sub
	return oa
}

func (oa ObjectAlias) ReplaceName(name string) Alias {
	oa.name = name
	return oa
}

func (oa ObjectAlias) ReplaceRootStruct(s *cst.Struct) Alias {
	oa.rootStruct = s
	return oa
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

func (g *AssignmentGenerator) Generate(srcAlias Alias) string {
	for _, srcField := range g.src.Fields {
		for _, dstField := range g.dst.Fields {
			if srcField.Name == dstField.Name {
				g.generateAssignmentSegment(srcAlias, srcField, dstField)
			}
		}
	}
	return g.writer.String()
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
func (g *AssignmentGenerator) generateBasicTypeAssignmentConvertFunc(srcAlias Alias, src cst.Field, dst cst.Field) {
	var (
		srcType   = src.Type.BaseType
		dstType   = dst.Type.BaseType
		aliasName = srcAlias.String()
	)
	// 数组类型进行基本数据类型转换时，需要取出ElementType
	if dst.Type.ElementType != nil {
		srcType = *src.Type.ElementType
		dstType = *dst.Type.ElementType
	}
	if srcType.Name == dstType.Name {
		if srcType.Star && !dstType.Star {
			// F float64    req.F *float64
			// F: *req.F,
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (i %s) { if %s && %s != nil { i = *%s } ; return i }()",
					dstType, statement, aliasName, aliasName)
			} else {
				g.print("func() (i %s) { return *%s }()",
					dstType, aliasName)
			}

		} else if !srcType.Star && dstType.Star {
			// F *float64   req.F float64
			// F: &req.F,
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (i %s) { if %s { i = &%s } ; return i }()",
					dstType, statement, aliasName)
			} else {
				g.print("func() (i %s) { return &%s }()",
					dstType, aliasName)
			}
		} else {
			// Message string    resp.Message string
			//Message: resp.Message,
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (i %s) { if %s { i = %s } ; return i }()",
					dstType, statement, aliasName)
			} else {
				g.print("func() (i %s) { return %s }()",
					dstType, aliasName)
			}
		}
	} else {
		if srcType.Star && !dstType.Star {
			// T int64  req.T *int
			// T: int64(*req.T),
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (i %s) { if %s { i = %s(*%s) } ; return i }()",
					dstType, statement, dstType.Name, aliasName)
			} else {
				g.print("func() (i %s) { return %s(*%s) }()",
					dstType, dstType.Name, aliasName)
			}
		} else if !srcType.Star && dstType.Star {
			// Y *int64    req.Y int
			// Y: func(i int) *int64 { return &i }(req.Y),
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (i %s) { if %s { k := %s(%s);i = &k } ; return i }()",
					dstType, statement, dstType.Name, aliasName)
			} else {
				g.print("func() (i %s) { k := %s(%s);i = &k ; return i }()",
					dstType, dstType.Name, aliasName)
			}
		} else {
			// Code int64  resp.Code int
			// Code: int64(resp.Code),
			if statement, isNeed := srcAlias.CheckNil(); isNeed {
				g.print("func() (i %s) { if %s { i = %s(%s) } ; return i }()",
					dstType, statement, dstType.Name, aliasName)
			} else {
				g.print("func() (i %s) { return %s(%s) }()",
					dstType, dstType.Name, aliasName)
			}
		}
	}
	return
}

// 生成结构体转换的方法体
func (g *AssignmentGenerator) generateStructTypeAssignmentConvertFunc(srcAlias Alias, src, dst cst.Field, srcStruct, dstStruct *cst.Struct) {
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
						g.generateAssignmentSegment(
							newAlias,
							srcField,
							dstField,
						)
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
	return
}

func (g *AssignmentGenerator) generateAssignmentSegment(srcAlias Alias, src cst.Field, dst cst.Field) {
	switch dst.Type.GoType {
	case cst.BasicType:
		g.print("%s: ", dst.Name)
		g.generateBasicTypeAssignmentConvertFunc(srcAlias.With(src.Name), src, dst)
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
		g.generateStructTypeAssignmentConvertFunc(srcAlias.With(src.Name), src, dst, srcStruct, dstStruct)
		g.println(",")
	case cst.ArrayType:
		switch dst.Type.ElementType.GoType {
		case cst.BasicType:
			// 基础类型相同的数组直接赋值，不需要转换
			// []string = []string, []int = []int ...
			if src.Type.Name == dst.Type.Name &&
				src.Type.Star == dst.Type.Star {
				g.println("%s: %s.%s,", dst.Name, srcAlias, src.Name)
				return
			}

			// 属于基础类型，但是互相转换的类型或者指针类型不对时
			// 生成方法转换, 例如[]int > []int64, []*int > []int

			// 从父级别名新增引用
			srcAlias = srcAlias.With(src.Name)
			g.println("%s: func(src %s) (dst %s) {", dst.Name, src.Type.String(), dst.Type.String())
			{
				g.println("dst = make( %s, len(src))", dst.Type.String())
				g.println("for i, v := range src{")
				{
					g.print("dst[i] = ")
					g.generateBasicTypeAssignmentConvertFunc(
						NewSimpleAlias("v"),
						src,
						dst,
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
				g.println("for i, v := range src{")
				{
					g.print("dst[i] = ")
					g.generateStructTypeAssignmentConvertFunc(
						srcAlias.ReplaceName("v"),
						src,
						dst,
						srcStruct,
						dstStruct,
					)
					g.println("")
				}
				g.println("}")
				g.println("return")
			}
			g.println("}(%s),", srcAlias)
		default:
			panic("unsupport type" + dst.Type.String())
		}
	case cst.MapType:
		switch dst.Type.ValueType.GoType {
		case cst.BasicType:
			// 基础类型相同的map直接赋值，不需要转换
			// map[string]string = map[string]string
			if src.Type.Name == dst.Type.Name &&
				src.Type.Star == dst.Type.Star {
				g.println("%s: %s.%s,", dst.Name, srcAlias, src.Name)
			}
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
					g.generateStructTypeAssignmentConvertFunc(
						srcAlias.ReplaceName("v"),
						src,
						dst,
						srcStruct,
						dstStruct,
					)
				}
				g.println("}")
				g.println("return")
			}
			g.println("}(%s),", srcAlias)
		default:
			panic("unsupport type" + dst.Type.String())
		}
	}
}

func (g *AssignmentGenerator) print(format string, a ...interface{}) {
	g.writer.WriteString(fmt.Sprintf(format, a...))
}

func (g *AssignmentGenerator) println(format string, a ...interface{}) {
	g.writer.WriteString(fmt.Sprintf(format+"\n", a...))
}
