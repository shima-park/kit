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

type AssignmentGenerator struct {
	cst           cst.ConcreteSyntaxTree
	pbcst         cst.ConcreteSyntaxTree
	writer        *bytes.Buffer
	src           *cst.Struct
	dst           *cst.Struct
	referenceType map[string]struct{} // key: [struct.Name or type.Name] val: struct{}{}
}

// TODO 两种别名处理，判断指针为空
type Alias interface {
	CheckNil() (statement string, isNeed bool)
	String() string
	IsStar() bool
	With(sub string) Alias
}

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

func (oa ObjectAlias) CheckNil() (statement string, isNeed bool) {
	aliases := strings.Split(oa.name, ".")
	concat := []string{}
	conditions := []string{}
	tempStruct := oa.rootStruct
	for i, alias := range aliases {
		if i == 0 {
			if oa.isStar {
				concat = append(concat, alias)
				conditions = append(conditions, fmt.Sprintf(" %s != nil ", strings.Join(concat, ".")))
			}
		} else {
			for _, field := range tempStruct.Fields {
				if field.Name == alias {
					pkg := oa.rootStruct.PackageName
					typeName := field.Type.Name
					typeStruct, found := findStruct(pkg, typeName, oa.csts...)
					if found {
						tempStruct = typeStruct
					} else {
						panic(fmt.Sprintf("not found struct(%s.%s)", pkg, typeName))
					}

					concat = append(concat, field.Name)

					if field.Type.Star {
						conditions = append(conditions, fmt.Sprintf(" %s != nil ", strings.Join(concat, ".")))
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

func (g *AssignmentGenerator) generateAssignmentSegment(srcAlias Alias, src cst.Field, dst cst.Field) {
	switch dst.Type.GoType {
	case cst.BasicType:
		if src.Type.Name == dst.Type.Name {
			if src.Type.Star && !dst.Type.Star {
				// F float64    req.F *float64
				// F: *req.F,
				if statement, isNeed := srcAlias.CheckNil(); isNeed {
					g.println("%s: func() (i %s) { if %s && %s.%s != nil { i = *%s.%s } ; return i }(),",
						dst.Name, dst.Type.String(), statement, srcAlias, src.Name, srcAlias, src.Name)
				} else {
					g.println("%s: func() (i %s) { return *%s.%s } (),",
						dst.Name, dst.Type.String(), srcAlias, src.Name)
				}

			} else if !src.Type.Star && dst.Type.Star {
				// F *float64   req.F float64
				// F: &req.F,
				if statement, isNeed := srcAlias.CheckNil(); isNeed {
					g.println("%s: func() (i %s) { if %s { i = &%s.%s } ; return i }(),",
						dst.Name, dst.Type.String(), statement, srcAlias, src.Name)
				} else {
					g.println("%s: func() (i %s) { return &%s.%s }(),",
						dst.Name, dst.Type.String(), srcAlias, src.Name)
				}
			} else {
				// Message string    resp.Message string
				//Message: resp.Message,
				if statement, isNeed := srcAlias.CheckNil(); isNeed {
					g.println("%s: func() (i %s) { if %s { i = %s.%s } ; return i }(),",
						dst.Name, dst.Type.String(), statement, srcAlias, src.Name)
				} else {
					g.println("%s: func() (i %s) { return %s.%s }(),",
						dst.Name, dst.Type.String(), srcAlias, src.Name)
				}
			}
		} else {
			if src.Type.Star && !dst.Type.Star {
				// T int64  req.T *int
				// T: int64(*req.T),
				if statement, isNeed := srcAlias.CheckNil(); isNeed {
					g.println("%s: func() (i %s) { if %s && %s.%s != nil { i = %s(*%s.%s) } ; return i }(),",
						dst.Name, dst.Type.String(), statement, srcAlias, src.Name, dst.Type.Name, srcAlias, src.Name)
				} else {
					g.println("%s: func() (i %s) { return %s(*%s.%s) }(),",
						dst.Name, dst.Type.String(), dst.Type.Name, srcAlias, src.Name)
				}
			} else if !src.Type.Star && dst.Type.Star {
				// Y *int64    req.Y int
				// Y: func(i int) *int64 { return &i }(req.Y),
				if statement, isNeed := srcAlias.CheckNil(); isNeed {
					g.println("%s: func() (i %s) { if %s { k := %s(%s.%s);i = &k } ; return i }(),",
						dst.Name, dst.Type.String(), statement, dst.Type.Name, srcAlias, src.Name)
				} else {
					g.println("%s: func() (i %s) { k := %s(%s.%s);i = &k ; return i }(),",
						dst.Name, dst.Type.String(), dst.Type.Name, srcAlias, src.Name)
				}
			} else {
				// Code int64  resp.Code int
				// Code: int64(resp.Code),
				if statement, isNeed := srcAlias.CheckNil(); isNeed {
					g.println("%s: func() (i %s) { if %s { i = %s(%s.%s) } ; return i }(),",
						dst.Name, dst.Type.String(), statement, dst.Type.Name, srcAlias, src.Name)
				} else {
					g.println("%s: func() (i %s) { return %s(%s.%s) }(),",
						dst.Name, dst.Type.String(), dst.Type.Name, srcAlias, src.Name)
				}
			}
		}

	case cst.ArrayType:
		switch dst.Type.ElementType.GoType {
		case cst.BasicType:
			if src.Type.Name == dst.Type.Name {
				if src.Type.Star == dst.Type.Star {
					g.println("%s: %s.%s,", dst.Name, srcAlias, src.Name)
					return
				}
			}
		default:
			srcStruct := g.findStruct(g.src.PackageName, src.Type.ElementType.Name)
			dstStruct := g.findStruct(g.dst.PackageName, dst.Type.ElementType.Name)

			src.Type.ElementType.X = srcStruct.PackageName
			dst.Type.ElementType.X = dstStruct.PackageName
			g.println("%s: func(src %s) (dst %s) {", dst.Name, src.Type.String(), dst.Type.String())
			g.println("dst = make( %s, len(src))", dst.Type.String())
			g.println("for i, v := range src{")
			g.print("dst[i] = ")

			if dst.Type.ElementType.Star {
				removeStarType := dst.Type.ElementType
				removeStarType.Star = false
				g.println("&%s{", removeStarType.String())
			} else {
				g.println("%s{", dst.Type.ElementType.String())
			}
			for _, srcField := range srcStruct.Fields {
				for _, dstField := range dstStruct.Fields {
					if srcField.Name == dstField.Name {
						g.generateAssignmentSegment(NewSimpleAlias("v"), srcField, dstField)
					}
				}
			}
			g.println("}")

			g.println("}")
			g.println("return")
			g.println("}(%s),", fmt.Sprintf("%s.%s", srcAlias, src.Name))
		}
	case cst.MapType:
		switch dst.Type.ValueType.GoType {
		case cst.BasicType:
			if src.Type.Name == dst.Type.Name &&
				src.Type.Star == dst.Type.Star {
				g.println("%s: %s.%s,", dst.Name, srcAlias, src.Name)
			}
		default:
			srcStruct := g.findStruct(g.src.PackageName, src.Type.ValueType.Name)
			dstStruct := g.findStruct(g.dst.PackageName, dst.Type.ValueType.Name)

			src.Type.ValueType.X = srcStruct.PackageName
			dst.Type.ValueType.X = dstStruct.PackageName
			g.println("%s: func(src %s) (dst %s) {", dst.Name, src.Type.String(), dst.Type.String())
			g.println("dst = make(%s, len(src))", dst.Type.String())
			g.println("for k, v := range src{")
			g.print("dst[k] = ")

			if dst.Type.ValueType.Star {
				removeStarType := dst.Type.ValueType
				removeStarType.Star = false
				g.println("&%s{", removeStarType.String())
			} else {
				g.println("%s{", dst.Type.ValueType.String())
			}

			for _, srcField := range srcStruct.Fields {
				for _, dstField := range dstStruct.Fields {
					if srcField.Name == dstField.Name {
						g.generateAssignmentSegment(NewSimpleAlias("v"), srcField, dstField)
					}
				}
			}
			g.println("}")

			g.println("}")
			g.println("return")
			//)return &i }(%s.%s),", dst.Name, src.Type.Name, dst.Type.String(), fmt.Sprintf("%s.%s", srcAlias, src.Name), src.Name)
			g.println("}(%s),", fmt.Sprintf("%s.%s", srcAlias, src.Name))
		}
	case cst.StructType:
		if src.Type.Name == dst.Type.Name {
			srcStruct := g.findStruct(g.src.PackageName, src.Type.Name)
			dstStruct := g.findStruct(g.dst.PackageName, dst.Type.Name)
			dst.Type.X = dstStruct.PackageName

			if dst.Type.Star {
				removeStarType := dst.Type
				removeStarType.Star = false
				g.println("%s: &%s{", dst.Name, removeStarType.String())
			} else {
				g.println("%s: %s{", dst.Name, dst.Type.String())
			}

			for _, srcField := range srcStruct.Fields {
				for _, dstField := range dstStruct.Fields {
					if srcField.Name == dstField.Name {
						g.generateAssignmentSegment(srcAlias.With(src.Name), srcField, dstField)
					}
				}
			}
			g.println("},")

		}

	}
}

func (g *AssignmentGenerator) print(format string, a ...interface{}) {
	g.writer.WriteString(fmt.Sprintf(format, a...))
}

func (g *AssignmentGenerator) println(format string, a ...interface{}) {
	g.writer.WriteString(fmt.Sprintf(format+"\n", a...))
}
