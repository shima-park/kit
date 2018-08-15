package generator

import (
	"fmt"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
)

type ReqAndResp struct {
	MethodName string
	Request    *cst.Struct
	Response   *cst.Struct
}

func GetRequestAndResponseList(cst cst.ConcreteSyntaxTree) []ReqAndResp {
	var rars []ReqAndResp
	for _, method := range cst.Interfaces()[0].Methods {
		var (
			rar = ReqAndResp{
				MethodName: method.Name,
			}
			foundReqOrResp bool
		)
		for _, param := range method.Params {
			if strings.HasSuffix(param.Type.Name, "Request") {
				strc, found := cst.StructMap()[cst.PackageName()][param.Type.Name]
				if found {
					//tc.requests = append(tc.requests, strc)
					rar.Request = strc
					foundReqOrResp = true
				}
				break
			}
		}

		for _, result := range method.Results {
			if strings.HasSuffix(result.Type.Name, "Response") {
				strc, found := cst.StructMap()[cst.PackageName()][result.Type.Name]
				if found {
					//tc.responses = append(tc.responses, strc)
					rar.Response = strc
					foundReqOrResp = true
				}
				break
			}
		}

		if !foundReqOrResp {
			continue
		}
		rars = append(rars, rar)
	}
	return rars
}

func GetReferenceStructMap(tree cst.ConcreteSyntaxTree, s *cst.Struct) map[string]*cst.Struct {
	referenceStructMap := map[string]*cst.Struct{}
	for _, field := range s.Fields {
		t := field.Type
		typ := t.BaseType
		if t.ElementType != nil {
			typ = *t.ElementType
		} else if t.ValueType != nil {
			typ = *t.ValueType
		}
		// 如果当前类型是struct 递归出所有组合的struct
		if typ.GoType != cst.StructType {
			continue
		}
		for _, structMap := range tree.StructMap() {
			strc, found := structMap[typ.Name]
			if found {
				_, found = referenceStructMap[strc.Name]
				if !found {
					referenceStructMap[strc.Name] = strc
					// 将循环的字段名和递归入口的类型名做判断
					// 防止死循环
					if field.Type.Name != t.Name {
						subStructMap := GetReferenceStructMap(tree, strc)
						MergeStructMap(subStructMap, referenceStructMap)
					}
				}
			}
		}
	}
	return referenceStructMap
}

func MergeStructMap(ms ...map[string]*cst.Struct) map[string]*cst.Struct {
	result := map[string]*cst.Struct{}
	copy := func(src, dst map[string]*cst.Struct) {
		for key, val := range src {
			_, found := dst[key]
			if !found {
				dst[key] = val
			}
		}
	}
	for _, m := range ms {
		copy(m, result)
	}
	return result
}

func FilterInterface(ifaces []cst.Interface, suffix string) (cst.Interface, error) {
	for _, iface := range ifaces {
		if strings.HasSuffix(iface.Name, suffix) {
			return iface, nil
		}
	}
	return cst.Interface{}, fmt.Errorf("No %s suffix service found", suffix)
}
