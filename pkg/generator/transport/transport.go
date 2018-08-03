package transport

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
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

func generateAssignmentSegment(dst *cst.Struct, srcs []gen.ReqAndResp, srcAlias string) string {
	src := findAssignmentStruct(dst, srcs)
	if dst != nil {
		return generateStructAssignmentSegment(src, dst, srcAlias)
	}
	return ""
}

type AssignmentGeneratorFactory struct {
	cst   cst.ConcreteSyntaxTree
	pbcst cst.ConcreteSyntaxTree
}

func NewAssignmentGeneratorFactory(cst cst.ConcreteSyntaxTree, pbcst cst.ConcreteSyntaxTree) *AssignmentGeneratorFactory {
	return &AssignmentGeneratorFactory{cst: cst, pbcst: pbcst}
}

func (g *AssignmentGeneratorFactory) Generate(dst *cst.Struct, srcs []gen.ReqAndResp, srcAlias string) string {
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

func (g *AssignmentGenerator) Generate(srcAlias string) string {
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

func (g *AssignmentGenerator) generateAssignmentSegment(srcAlias string, src cst.Field, dst cst.Field) {

	switch dst.Type.GoType {
	case cst.BasicType:
		if src.Type.Name == dst.Type.Name {
			if src.Type.Star && !dst.Type.Star {
				// F float64    req.F *float64
				// F: *req.F,
				g.println("%s: func() (i %s) { if %s != nil { i = *%s.%s } ; return i }(),", dst.Name, dst.Type.String(), srcAlias, srcAlias, src.Name)
			} else if !src.Type.Star && dst.Type.Star {
				// F *float64   req.F float64
				// F: &req.F,
				g.println("%s: &%s.%s,", dst.Name, srcAlias, src.Name)
			} else {
				// Message string    resp.Message string
				//Message: resp.Message,
				g.println("%s: %s.%s,", dst.Name, srcAlias, src.Name)
			}
		} else {
			if src.Type.Star && !dst.Type.Star {
				// T int64  req.T *int
				// T: int64(*req.T),
				g.println("%s: %s(*%s.%s),", dst.Name, dst.Type.Name, srcAlias, src.Name)
			} else if !src.Type.Star && dst.Type.Star {
				// Y *int64    req.Y int
				// Y: func(i int) *int64 { return &i }(req.Y),
				g.println("%s: func() %s { i := %s(%s.%s); return &i }(),", dst.Name, dst.Type.String(), dst.Type.Name, srcAlias, src.Name)
			} else {
				// Code int64  resp.Code int
				// Code: int64(resp.Code),
				g.println("%s: %s(%s.%s),", dst.Name, dst.Type.Name, srcAlias, src.Name)
			}
		}

	case cst.ArrayType:
		switch dst.Type.ElementType.GoType {
		case cst.BasicType:
			if src.Type.Name == dst.Type.Name &&
				src.Type.Star == dst.Type.Star {
				g.println("%s: %s.%s,", dst.Name, srcAlias, src.Name)
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
						g.generateAssignmentSegment("v", srcField, dstField)
					}
				}
			}
			g.println("}")

			g.println("}")
			g.println("return")
			//)return &i }(%s.%s),", dst.Name, src.Type.Name, dst.Type.String(), fmt.Sprintf("%s.%s", srcAlias, src.Name), src.Name)
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
						g.generateAssignmentSegment("v", srcField, dstField)
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
						g.generateAssignmentSegment(fmt.Sprintf("%s.%s", srcAlias, src.Name), srcField, dstField)
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

func generateStructAssignmentSegment(src *cst.Struct, dst *cst.Struct, srcAlias string) string {
	buff := bytes.NewBufferString("")
	for _, dstField := range dst.Fields {
		for _, srcField := range src.Fields {
			if srcField.Name != dstField.Name {
				continue
			}
			generateFieldAssignmentSegment(buff, srcField, dstField, srcAlias)
		}
	}

	return buff.String()
}

func generateFieldAssignmentSegment(buff io.Writer, srcField cst.Field, dstField cst.Field, srcAlias string) {
	logrus.Info(
		cst.EqualStructField(srcField, dstField),
		cst.IsBasicType(srcField.Type.Name) &&
			cst.IsBasicType(dstField.Type.Name),
		dstField.Name, srcAlias, srcField.Name,
		"----type----",
		dstField.Type.String(), srcField.Type.String(),
	)
	if cst.EqualStructField(srcField, dstField) {
		buff.Write([]byte(fmt.Sprintf("%s: %s.%s,\n", dstField.Name, srcAlias, srcField.Name)))
		return
	}
	if cst.IsBasicType(srcField.Type.Name) &&
		cst.IsBasicType(dstField.Type.Name) {
		buff.Write([]byte(fmt.Sprintf("%s: %s(%s.%s),\n", dstField.Name, dstField.Type.String(), srcAlias, srcField.Name)))
		return
	}
}
