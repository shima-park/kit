package generator

import (
	"html/template"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
)

type TemplateConfig interface {
	Funcs() template.FuncMap
	Data() interface{}
}

var NoopTemplateConifg = noopTemplateConifg{}

type noopTemplateConifg struct {
}

func (n noopTemplateConifg) Funcs() template.FuncMap {
	return map[string]interface{}{}
}
func (n noopTemplateConifg) Data() interface{} {
	return map[string]interface{}{}
}

type templateConfig struct {
	cst                  cst.ConcreteSyntaxTree
	ifaceMethods         []cst.Method
	ifaceName            string
	implStruct           *cst.Struct
	implMethods          []cst.Method
	requests             []*cst.Struct
	responses            []*cst.Struct
	requestsAndResponses []*ReqAndResp
	referenceStruct      []*cst.Struct
}

func NewTemplateConfig(cst cst.ConcreteSyntaxTree) TemplateConfig {
	tc := &templateConfig{
		cst: cst,
	}

	tc.build()

	return tc
}

func (tc *templateConfig) build() {
	tc.buildInterfaceInfo()

	tc.buildImplementationInfo()

	tc.buildRequestAndResponseInfo()
}

type ReqAndResp struct {
	Request  *cst.Struct
	Response *cst.Struct
}

func (tc *templateConfig) buildInterfaceInfo() {
	for _, iface := range tc.cst.Interfaces() {
		tc.ifaceName = iface.Name
		for _, method := range iface.Methods {
			tc.ifaceMethods = append(tc.ifaceMethods, method)
		}
		// TODO 目前只做一个文件对应生成一个endpoint，不考虑多interface
		break
	}
}

func (tc *templateConfig) buildImplementationInfo() {
	for _, iface := range tc.cst.Interfaces() {
		for _, strc := range tc.cst.StructMap()[tc.cst.PackageName()] {
			if !cst.IsInterfaceImplementation(iface, strc) {
				continue
			}
			tc.implStruct = strc
			return
		}
	}

	if tc.implStruct == nil {
		panic("Service interface has no implementation struct")
	}
}

func (tc *templateConfig) buildRequestAndResponseInfo() {
	if tc.implStruct == nil {
		panic("Service interface has no implementation struct")
	}

	for _, method := range tc.implStruct.Methods {

		tc.implMethods = append(tc.implMethods, method)

		var (
			rar            ReqAndResp
			foundReqOrResp bool
		)
		for _, param := range method.Params {
			// TODO 实现类判断
			if strings.HasSuffix(param.Type.Name, "Request") {
				strc, found := tc.cst.StructMap()[tc.cst.PackageName()][param.Type.Name]
				if found {
					tc.requests = append(tc.requests, strc)
					rar.Request = strc
					foundReqOrResp = true
				}
				break
			}
		}

		for _, result := range method.Results {
			// TODO 实现类判断
			if strings.HasSuffix(result.Type.Name, "Response") {
				strc, found := tc.cst.StructMap()[tc.cst.PackageName()][result.Type.Name]
				if found {
					tc.responses = append(tc.responses, strc)
					rar.Response = strc
					foundReqOrResp = true
				}
				break
			}
		}

		if !foundReqOrResp {
			continue
		}
		tc.requestsAndResponses = append(tc.requestsAndResponses, &rar)
	}
}

func (tc *templateConfig) Data() interface{} {
	return map[string]interface{}{
		"PackageName":           tc.cst.PackageName(),
		"ImportPath":            "",
		"InterfaceName":         tc.ifaceName,
		"InterfaceMethods":      tc.ifaceMethods,
		"Implementation":        tc.implStruct,
		"ImplementationMethods": tc.implMethods,
		"Requests":              tc.requests,
		"Responses":             tc.responses,
		"RequestsAndResponses":  tc.requestsAndResponses,
	}
}

func (d *templateConfig) Funcs() template.FuncMap {
	return map[string]interface{}{
		"ToCamelCase":              ToCamelCase,
		"ToLowerFirstCamelCase":    ToLowerFirstCamelCase,
		"ToLowerSnakeCase":         ToLowerSnakeCase,
		"ToUpperFirst":             ToUpperFirst,
		"JoinFieldsByComma":        cst.NewFieldsToString(","),
		"JoinFieldsByLineBreak":    cst.NewFieldsToString("\n"),
		"JoinFieldKeysByComma":     cst.NewFieldsKeyToString(","),
		"JoinFieldKeysByLineBreak": cst.NewFieldsKeyToString("\n"),
	}
}
