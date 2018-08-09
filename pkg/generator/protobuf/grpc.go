package protobuf

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
)

type ProtobufGenerator struct {
	cst           cst.ConcreteSyntaxTree
	opts          Options
	referenceType map[string]struct{} // key: [struct.Name or type.Name] val: struct{}{}
}

func NewProtobufGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
	options := newOptions(opts...)

	return &ProtobufGenerator{
		cst:           t,
		opts:          options,
		referenceType: map[string]struct{}{},
	}
}

func (g *ProtobufGenerator) Generate() error {
	baseServiceName := service.GetBaseServiceName(g.cst.PackageName(), g.opts.serviceSuffix)
	protobufPath := utils.GetProtobufFilePath(baseServiceName)
	protobufPackageName := filepath.Base(protobufPath)
	w := NewSugerWriter(g.opts.writer)
	w.P(`syntax = "proto3";`)
	w.P(``)
	w.P("package %s;", protobufPackageName)
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

func (g *ProtobufGenerator) isUseStruct(structName string) bool {
	_, found := g.referenceType[structName]
	return found
}

func (g *ProtobufGenerator) generateInterface(i cst.Interface) {
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

func (g *ProtobufGenerator) generateServiceMethod(method cst.Method) {
	w := NewSugerWriter(g.opts.writer)
	w.P(`rpc %s (`, method.Name)
	g.generateServiceMethodFields(method.Params)
	w.P(`)`)
	w.P(` returns (`)
	g.generateServiceMethodFields(method.Results)
	w.P(`) {}`)
	w.P(``)
}

func (g *ProtobufGenerator) generateServiceMethodFields(fields []cst.Field) {
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

func (g *ProtobufGenerator) getGrpcType(t cst.Type) (grpcType string, ignore bool) {
	grpcType, found := g.GoType2GrpcType(t)
	if !found {
		pkg := g.cst.PackageName()
		// 尝试从type所在的包查找
		if t.X != "" {
			pkg = t.X
		}

		grpcType, found := g.findStructInASTStructMap(pkg, t.Name)
		if !found {
			panic(fmt.Sprintf("Not found (%+v) in grpc type mapping(pkg:%s) and ast StructMap", t, pkg))
		}
		return grpcType, false
	}
	return grpcType, false
}

func (g *ProtobufGenerator) recursiveFieldType(t cst.Type) {
	typ := t.BaseType
	if t.ElementType != nil {
		typ = *t.ElementType
	} else if t.ValueType != nil {
		typ = *t.ValueType
	}
	// 如果当前类型是struct 递归出所有组合的struct
	if typ.GoType != cst.StructType {
		return
	}
	for _, structMap := range g.cst.StructMap() {
		strc, found := structMap[typ.Name]
		if found {
			_, found = g.referenceType[strc.Name]
			if !found {
				g.referenceType[strc.Name] = struct{}{}
				for _, field := range strc.Fields {
					// 将循环的字段名和递归入口的类型名做判断
					// 防止死循环
					if field.Type.Name != t.Name {
						g.recursiveFieldType(field.Type)
					}
				}
			}
		}
	}
}

func (g *ProtobufGenerator) generateMessage(strc *cst.Struct) {
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

func (g *ProtobufGenerator) checkPBTag(strc *cst.Struct) {
	var (
		useTagCount int
		seqMap      = map[int]cst.Field{}    // key: seq value: field
		nameMap     = map[string]cst.Field{} // key: name value: field
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
			useTagCount++

			pbTags := strings.Split(pbTagStr, ",")
			for _, pbTag := range pbTags {
				switch {
				case strings.HasPrefix(pbTag, "seq="):
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
						panic(fmt.Sprintf("StructName:%s Field:%s and Field:%s have the same tag:seq(%d)\n  Field:%s %s %s\n  Field:%s %s %s",
							strc.Name, field.Name, field2.Name, seq,
							field.Name, field.Pos, field.Tag,
							field2.Name, field2.Pos, field2.Tag,
						))
					}

				case strings.HasPrefix(pbTag, "type="):

				case strings.HasPrefix(pbTag, "name="):
					name := pbTag[strings.Index(pbTag, "=")+1:]
					field2, found := nameMap[name]
					if !found {
						nameMap[name] = field
					} else {
						panic(fmt.Sprintf("StructName:%s Field:%s and Field:%s have the same tag:name(%s)\n  Field:%s %s %s\n  Field:%s %s %s",
							strc.Name, field.Name, field2.Name, name,
							field.Name, field.Pos, field.Tag,
							field2.Name, field2.Pos, field2.Tag,
						))
					}
				}
			}
		}
	}

	// 使用了pb这个tag但是并没有给所有的字段加上，这种情况没办法增加序列号或者检查命名冲突
	if useTagCount > 0 && useTagCount != len(strc.Fields) {
		panic(fmt.Sprintf("If you use the \"pb\" tag you must set for(StructName:%s) all fields\n %s", strc.Name, strc.Position.String()))
	}
}

func (g *ProtobufGenerator) findStructInASTStructMap(pkg, structName string) (string, bool) {
	if strc, found := g.cst.StructMap()[pkg][structName]; found {
		return strc.Name, true
	}

	return "", false
}

func (g *ProtobufGenerator) GoType2GrpcType(t cst.Type) (grpcType string, found bool) {
	goType := strings.TrimSpace(t.Name)
	switch t.GoType {
	case cst.BasicType:
		return GoBasicType2GrpcType(goType)
	case cst.ArrayType:
		// grpc 没有单个byte的类型，特殊判断一下
		if goType == "[]byte" {
			return "bytes", true
		}

		switch t.ElementType.GoType {
		case cst.BasicType:
			grpcType, found = GoBasicType2GrpcType(t.ElementType.Name)
			if !found {
				return "", false
			}
		case cst.StructType:
			grpcType = t.ElementType.Name
			found = true
		default:
			panic("Unsupport grpc item of array:" + t.ElementType.Name)
		}

		return "repeated " + grpcType, true
	case cst.MapType:
		// TODO Key in map field cannot be float/doubl, bytes or message types.
		// protobuf的key类型不能为float/doubl, bytes or message

		var keyType string
		switch t.KeyType.GoType {
		case cst.BasicType:
			keyType, found = GoBasicType2GrpcType(t.KeyType.Name)
			if !found {
				return "", false
			}
		default:
			panic("Unsupport grpc type of key of map:" + t.KeyType.Name)
		}

		var valueType string
		switch t.ValueType.GoType {
		case cst.BasicType:
			valueType, found = GoBasicType2GrpcType(t.ValueType.Name)
			if !found {
				return "", false
			}
		case cst.StructType:
			valueType = t.ValueType.Name
			found = true
		default:
			panic("Unsupport grpc value of key of map:" + t.KeyType.Name)
		}

		return fmt.Sprintf("map<%s, %s>", keyType, valueType), true
	case cst.StructType:
		return t.Name, true
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
