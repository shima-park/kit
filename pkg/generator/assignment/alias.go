package assignment

import (
	"fmt"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
)

type Alias interface {
	CheckNil() (statement string, isNeed bool) // checknil的表达式，是否需要检查
	String() string
	IsStar() bool                  // rootStruct是否是指针类型
	With(sub string) Alias         // 别名叠加 1.A 2.A.With(B) 3.A.B
	ReplaceName(name string) Alias // 别名替换 1.A 2.A.ReplaceName(B) 3.B
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

// 对象别名实现，创建时传入根别名，是否是指针类型，及别名的对象数据结构
// 往下传递时，通过With增加 例如当alias.name=resp 向下引用 resp.Data = alias.With("Data")
// 生成检查空指针方法时，通过遍历rootStruct字段和分割的alias对应的字段，
// 通过csts数组查找Struct并获取其类型，判断是否需要生成检查空指针方法
type ObjectAlias struct {
	name       string                   // 别名对象当前的名字
	csts       []cst.ConcreteSyntaxTree // 所涉及到的go源码文件的实际结构
	rootStruct *cst.Struct              // 对象别名本身的数据结构
	isStar     bool                     // rootStruct是否是指针类型
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
				// 获取到字段值存储的数据类型
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
