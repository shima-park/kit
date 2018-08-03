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

func FilterInterface(ifaces []cst.Interface, suffix string) (cst.Interface, error) {
	for _, iface := range ifaces {
		if strings.HasSuffix(iface.Name, suffix) {
			return iface, nil
		}
	}
	return cst.Interface{}, fmt.Errorf("No %s suffix service found", suffix)
}
