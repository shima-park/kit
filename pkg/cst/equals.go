package cst

import "fmt"

func IsInterfaceImplementation(iface Interface, strc *Struct) bool {
	for _, ifaceMethod := range iface.Methods {
		var equal bool
		for _, strcMethod := range strc.Methods {
			fmt.Println("--------", iface.Name, strc.Name, ifaceMethod.Name, strcMethod.Name, EqualMethod(ifaceMethod, strcMethod))
			if EqualMethod(ifaceMethod, strcMethod) {
				equal = true
				break
			}
		}
		if !equal {
			return false
		}
	}
	return true
}

func EqualMethod(expect, actual Method) bool {
	if expect.Name != actual.Name {
		return false
	}

	for _, expectParam := range expect.Params {
		var equal bool
		fmt.Println("-----", expect.Name, expect.Params)
		fmt.Println("-----", actual.Name, actual.Params)
		for _, actualParam := range actual.Params {
			if EqualMethodField(expectParam, actualParam) {
				equal = true
				break
			}
		}
		if !equal {
			return false
		}
	}

	for _, expectResult := range expect.Results {
		var equal bool
		for _, actualResult := range actual.Results {
			if EqualMethodField(expectResult, actualResult) {
				equal = true
				break
			}
		}
		if !equal {
			return false
		}
	}
	return true
}

func EqualMethodField(expect, actual Field) bool {
	// 比较方法字段是否相同只需要比较参数类型和顺序是否相同
	// 不需要判断参数的名字是否相同

	if !EqualType(expect.Type, actual.Type) {
		return false
	}

	return true
}

func EqualStructField(expect, actual Field) bool {
	if expect.Name != actual.Name {
		return false
	}

	if !EqualType(expect.Type, actual.Type) {
		return false
	}

	return true
}

func EqualType(expect, actual Type) bool {
	if expect.GoType != actual.GoType {
		return false
	}

	if expect.Name != actual.Name {
		return false
	}

	if expect.Star != actual.Star {
		return false
	}

	return true
}
