package grpc

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
)

type GRPCGenerator struct {
	cst           cst.ConcreteSyntaxTree
	template      *template.Template
	opts          Options
	referenceType map[string]struct{} // key: [struct.Name or type.Name] val: struct{}{}
}

func NewGRPCGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.serviceNameNormalizer == nil {
		options.serviceNameNormalizer = gen.NoopNormalizer
	}

	if options.fieldNameNormalizer == nil {
		options.fieldNameNormalizer = gen.NoopNormalizer
	}

	if options.typeFilter == nil {
		options.typeFilter = gen.DefaultTypeFilter
	}

	if options.structFilter == nil {
		options.structFilter = gen.DefaultStructFilter
	}

	if options.writer == nil {
		options.writer = gen.DefaultWriter
	}

	return &GRPCGenerator{
		cst:           t,
		opts:          options,
		referenceType: map[string]struct{}{},
	}
}

func (g *GRPCGenerator) Generate() error {
	w := NewSugerWriter(g.opts.writer)
	w.P(`syntax = "proto3";`)
	w.P(``)
	w.P("package %s;", g.cst.PackageName())
	w.P(``)

	for _, i := range g.cst.Interfaces() {
		g.generateInterface(i)
	}

	for i, strc := range g.cst.Structs() {
		// 跳过制定过滤的struct 和 未使用的struct
		if strc == nil {
			panic(i)
		}
		if g.opts.structFilter(strc) &&
			!g.isUseStruct(strc.Name) {
			continue
		}

		g.generateMessage(strc)
	}

	return nil
}

func (g *GRPCGenerator) isUseStruct(structName string) bool {
	_, found := g.referenceType[structName]
	return found
}

func (g *GRPCGenerator) generateInterface(i cst.Interface) {
	w := NewSugerWriter(g.opts.writer)
	serviceName := g.opts.serviceNameNormalizer.Normalize(i.Name)
	w.P(`service %s {`, serviceName)
	w.P(``)
	for _, method := range i.Methods {
		g.generateServiceMethod(method)
	}
	w.P(`}`)
	w.P(``)
}

func (g *GRPCGenerator) generateServiceMethod(method cst.Method) {
	w := NewSugerWriter(g.opts.writer)
	w.P(`rpc %s (`, method.Name)
	g.generateServiceMethodFields(method.Params)
	w.P(`)`)
	w.P(` returns (`)
	g.generateServiceMethodFields(method.Results)
	w.P(`) {}`)
	w.P(``)
}

func (g *GRPCGenerator) generateServiceMethodFields(fields []cst.Field) {
	w := NewSugerWriter(g.opts.writer)
	for _, field := range fields {
		if g.opts.typeFilter(field.Type) {
			continue
		}

		if field.Type.GoType == cst.BasicType {
			panic(fmt.Sprintf("gRPC Request parameters unsupprt %s type(go type name:%s)", field.Type.GoType, field.Type.Name))
		}

		grpcType, ignore := g.getGrpcType(field.Type)
		if ignore {
			continue
		}

		g.recursiveFieldType(field.Type)

		w.P(`%s`, grpcType)
		// TODO 提示gRPC参数不能超过1位
		break
	}
}

func (g *GRPCGenerator) recursiveFieldType(t cst.Type) {
	// 如果当前类型是struct 递归出所有组合的struct
	if t.GoType != cst.StructType {
		return
	}
	for _, structMap := range g.cst.StructMap() {
		strc, found := structMap[t.Name]
		if found {
			_, found = g.referenceType[strc.Name]
			if found {
				for _, field := range strc.Fields {
					// 将循环的字段名和递归入口的类型名做判断
					// 防止死循环
					if field.Type.Name != t.Name {
						g.recursiveFieldType(field.Type)
					}
				}
			} else {
				g.referenceType[strc.Name] = struct{}{}
			}
		}
	}
}

func (g *GRPCGenerator) generateMessage(strc *cst.Struct) {
	w := NewSugerWriter(g.opts.writer)
	w.P(`message %s {`, strc.Name)
	w.P(``)

	g.checkPBTag(strc)

	for i, field := range strc.Fields {
		var (
			fieldName = field.Name
			seq       = i + 1
			grpcType  string
			ignore    bool
			tag       = reflect.StructTag(field.Tag)
			pbTagStr  = tag.Get("pb")
		)

		grpcType, ignore = g.getGrpcType(field.Type)
		if ignore {
			continue
		}

		// protobuf本身的tag(protobuf)中的数据类型不能直接作为proto中的数据类型使用
		// 能使用的仅序列号和字段名
		// 在这里自定义新增了一个tag(pb)用作name,seq,type的重定义
		// TODO 设置seq必须所有字段设置，否则会出现seq不唯一的情况
		if field.Tag != "" && pbTagStr != "" {
			pbTags := strings.Split(pbTagStr, ",")
			for _, pbTag := range pbTags {
				switch {
				case strings.HasPrefix(pbTag, "name="):
					fieldName = pbTag[strings.Index(pbTag, "=")+1:]
				case strings.HasPrefix(pbTag, "seq="):
					seqStr := pbTag[strings.Index(pbTag, "=")+1:]
					seq, _ = strconv.Atoi(seqStr)
				case strings.HasPrefix(pbTag, "type="):
					grpcType = pbTag[strings.Index(pbTag, "=")+1:]
				}
			}
		}

		w.P(`%s %s = %d;`, grpcType, fieldName, seq)
		w.P(``)
	}
	w.P(`}`)
	w.P(``)
}

func (g *GRPCGenerator) checkPBTag(strc *cst.Struct) {
	var (
		useTagSeq bool
		seqMap    = map[int]cst.Field{} // key: seq value: field
	)
	for _, field := range strc.Fields {
		var (
			tag      = reflect.StructTag(field.Tag)
			pbTagStr = tag.Get("pb")
		)

		// protobuf本身的tag(protobuf)中的数据类型不能直接作为proto中的数据类型使用
		// 能使用的仅序列号和字段名
		// 在这里自定义新增了一个tag(pb)用作name,seq,type的重定义
		// 设置seq必须所有字段设置，否则会出现seq不唯一的情况
		if field.Tag != "" && pbTagStr != "" {
			pbTags := strings.Split(pbTagStr, ",")
			for _, pbTag := range pbTags {
				switch {
				case strings.HasPrefix(pbTag, "seq="):
					if !useTagSeq {
						useTagSeq = true
					}

					seqStr := pbTag[strings.Index(pbTag, "=")+1:]
					seq, err := strconv.Atoi(seqStr)
					if err != nil {
						panic(fmt.Sprintf("Unsupport grpc pb StructName:%s Field:%s tag(%s) error(%v)\n %s",
							strc.Name, field.Name, pbTag, err, field.Pos))
					}

					field2, found := seqMap[seq]
					if !found {
						seqMap[seq] = field
					} else {
						panic(fmt.Sprintf("StructName:%s Field:%s and Field:%s have the same seq(%d)\n  Field:%s %s \n  Field:%s %s",
							strc.Name, field.Name, field2.Name, seq,
							field.Name, field.Pos,
							field2.Name, field2.Pos,
						))
					}
				case strings.HasPrefix(pbTag, "type="):

				}
			}
		} else {
			if useTagSeq {
				panic(fmt.Sprintf("If you use the \"seq\" tag you must set for(StructName:%s) all fields\n %s", strc.Name, strc.Pos))
			}
		}
	}
}

func (g *GRPCGenerator) getGrpcType(t cst.Type) (grpcType string, ignore bool) {
	grpcType, found := g.GoType2GrpcType(t)
	if !found {
		pkg := g.cst.PackageName()
		// 尝试从type所在的包查找
		if t.X != "" {
			pkg = t.X
		}

		grpcType, found := g.findStructInASTStructMap(pkg, t.Name)
		if !found {
			switch t.GoType {
			case cst.StructType:
				panic(fmt.Sprintf("Not found (%+v) in grpc type mapping(pkg:%s) and ast StructMap", t, pkg))
			}
			return grpcType, true
		}
		return grpcType, false
	}
	return grpcType, false
}

func (g *GRPCGenerator) findStructInASTStructMap(pkg, structName string) (string, bool) {
	if strc, found := g.cst.StructMap()[pkg][structName]; found {
		// 存储接口定义中使用过的结构类型，方便判断后面message中是否生成
		_, found = g.referenceType[structName]
		if !found {
			g.referenceType[structName] = struct{}{}
		}
		return strc.Name, true
	}

	return "", false
}

func (g *GRPCGenerator) GoType2GrpcType(t cst.Type) (grpcType string, found bool) {
	goType := strings.TrimSpace(t.Name)
	switch t.GoType {
	case cst.BasicType:
		return GoBasicType2GrpcType(goType)
	case cst.ArrayType:
		// grpc 没有单个byte的类型，特殊判断一下
		if goType == "[]byte" {
			return "bytes", true
		}

		var ident string
		if strings.HasPrefix(goType, "[]") {
			ident = goType[2:]
		} else if strings.HasPrefix(goType, "*[]") {
			ident = goType[3:]
		}

		if strings.Contains(ident, ".") {
			ss := strings.Split(ident, ".")
			pkg := strings.Trim(ss[0], "*")
			structName := ss[1]
			grpcType, found = g.findStructInASTStructMap(pkg, structName)
			if !found {
				return "", false
			}
		} else {
			structName := strings.Trim(ident, "*")
			grpcType, found = GoBasicType2GrpcType(structName)
			if !found {
				grpcType, found = g.findStructInASTStructMap(g.cst.PackageName(), structName)
			}
		}
		return "repeated " + grpcType, found
	case cst.MapType:
		quoteStart := strings.Index(goType, "[")
		quoteEnd := strings.Index(goType, "]")
		keyStr := goType[quoteStart+1 : quoteEnd]
		keyType, found := GoBasicType2GrpcType(keyStr)
		if !found {
			if strings.Contains(keyStr, ".") {
				ss := strings.Split(keyStr, ".")
				pkg := strings.Trim(ss[0], "*")
				structName := ss[1]
				keyType, found = g.findStructInASTStructMap(pkg, structName)
				if !found {
					return "", false
				}
			} else {
				keyType, found = g.findStructInASTStructMap(g.cst.PackageName(), keyStr)
				if !found {
					return "", false
				}
			}
		}

		valStr := goType[quoteEnd+1:]
		valueType, found := GoBasicType2GrpcType(valStr)
		if !found {
			if strings.Contains(valStr, ".") {
				ss := strings.Split(valStr, ".")
				pkg := strings.Trim(ss[0], "*")
				structName := ss[1]
				valueType, found = g.findStructInASTStructMap(pkg, structName)
				if !found {
					return "", false
				}
			} else {
				structName := strings.Trim(valStr, "*")
				valueType, found = g.findStructInASTStructMap(g.cst.PackageName(), structName)
				if !found {
					return "", false
				}
			}
		}

		return fmt.Sprintf("map<%s, %s>", keyType, valueType), true
	case cst.StructType:

	case cst.CrossProtocolUnsupportType:
		panic(fmt.Sprintf("This type(%s %s) is unsupport cross protocol", t.Name, t.GoType))
	}

	return "", false
}

func GoBasicType2GrpcType(t string) (grpcType string, found bool) {
	goType := strings.TrimSpace(t)
	switch goType {
	case "float64":
		return "double", true
	case "float32":
		return "float", true
	case "int32":
		return "int32", true
	case "int", "int64":
		return "int64", true
	case "uint32":
		return "uint32", true
	case "uint64":
		return "uint64", true
	case "bool":
		return "bool", true
	case "string":
		return "string", true
	}
	return "", false
}
