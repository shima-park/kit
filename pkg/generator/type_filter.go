package generator

import "ezrpro.com/micro/kit/pkg/cst"

type TypeFilter func(t cst.Type) bool

func NoopTypeFilter(t cst.Type) bool {
	return false
}

func DefaultTypeFilter(t cst.Type) bool {
	switch t.String() {
	case "context.Context", "error":
		return true
	}

	return false
}
