package cst

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"ezrpro.com/micro/kit/pkg/utils"
)

type concreteSyntaxTree struct {
	opts Options
	fset *token.FileSet
	file *ast.File

	packagePath string // 包名   ./pkg/addservice/addservice.go:1:1
	packageName string // 包路径 addservice

	imports    []Import
	interfaces []Interface
	vars       []Var
	consts     []Constant

	// key: package
	// val: map[key: structName] value : struct
	structMap map[string]map[string]*Struct
	typeMap   map[string]Type // key: typeName value: Type

	methods []Method

	// key: import path e.g. github.com/xxx/xxx
	// val: struct{}
	parsedReferencePackageMap map[string]struct{}
}

func NewConcreteSyntaxTree(fset *token.FileSet, file *ast.File, opts ...Option) ConcreteSyntaxTree {
	return newConcreteSyntaxTree(fset, file, opts...)
}

func newConcreteSyntaxTree(fset *token.FileSet, file *ast.File, opts ...Option) *concreteSyntaxTree {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.fieldNameFilter == nil {
		options.fieldNameFilter = DefaultFieldNameFilter
	}

	cst := &concreteSyntaxTree{
		opts:                      options,
		fset:                      fset,
		file:                      file,
		structMap:                 make(map[string]map[string]*Struct),
		typeMap:                   make(map[string]Type),
		parsedReferencePackageMap: make(map[string]struct{}),
	}

	return cst
}

func (t *concreteSyntaxTree) PackageName() string {
	if t.opts.packageName != "" {
		return t.opts.packageName
	}
	return t.packageName
}

func (t *concreteSyntaxTree) Imports() []Import {
	return t.imports
}

func (t *concreteSyntaxTree) Interfaces() []Interface {
	return t.interfaces
}

func (t *concreteSyntaxTree) Types() []Type {
	var typeNames = make([]string, len(t.typeMap))
	for typeName, _ := range t.typeMap {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)

	var types = make([]Type, len(typeNames))
	for i, typeName := range typeNames {
		types[i] = t.typeMap[typeName]
	}
	return types
}

func (t *concreteSyntaxTree) Structs() []*Struct {
	var structNames []string
	for pkg, _ := range t.structMap {
		for structName, _ := range t.structMap[pkg] {
			structNames = append(structNames, structName)
		}
	}

	sort.Strings(structNames)

	var structs = make([]*Struct, len(structNames))
	for i, structName := range structNames {
		for pkg, _ := range t.structMap {
			strc, found := t.structMap[pkg][structName]
			if found {
				structs[i] = strc
				break
			}
		}
	}
	return structs
}

func (t *concreteSyntaxTree) Methods() []Method {
	return t.methods
}

func (t *concreteSyntaxTree) Vars() []Var {
	return t.vars
}

func (t *concreteSyntaxTree) Consts() []Constant {
	return t.consts
}

func (t *concreteSyntaxTree) StructMap() map[string]map[string]*Struct {
	return t.structMap
}

func (t *concreteSyntaxTree) Parse() error {
	t.packagePath = t.fset.Position(t.file.Package).Filename
	t.packageName = t.file.Name.Name

	var err error
	for _, decl := range t.file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			switch gen.Tok {
			case token.IMPORT:
				t.parseImportSpec(gen.Specs)
			case token.TYPE:
				t.parseTypeSpec(gen.Specs)
			case token.VAR:
				t.parseVars(gen.Specs)
			case token.CONST:
				t.parseConst(gen.Specs)
			}

		} else if gen, ok := decl.(*ast.FuncDecl); ok {
			t.parseFuncDecl(gen)
		}
	}
	return err
}

func (t *concreteSyntaxTree) parseConst(specs []ast.Spec) {
	for _, sp := range specs {
		vsp, ok := sp.(*ast.ValueSpec)
		if !ok {
			panic("Var spec is not ValueSpec type, odd, skipping")
		}

		for i, ident := range vsp.Names {
			v := Constant{
				Name: ident.Name,
			}

			if vsp.Type != nil {
				v.Type = t.getFieldType(vsp.Type, "")
			}

			if len(vsp.Values) > 0 {
				switch vt := vsp.Values[i].(type) {
				case *ast.BasicLit:
					v.Value = vt.Value
					if vsp.Type == nil {
						// TODO
						// var定义时没有设置类型的，简陋推断
						// e.g. var i = 1
						v.Type = getTypeByValue(vt.Value)
					}
				case *ast.UnaryExpr:
					switch xt := vt.X.(type) {
					case *ast.CompositeLit:
						v.Type = t.getFieldType(xt.Type, "")
					}

					//TODO
					// case *ast.BinaryExpr:
					// e.g. 1<<31 - 1 这种表达式类型
				}
				if v.Value == nil {
					fst := token.NewFileSet()
					bt := bytes.NewBufferString("")
					err := format.Node(bt, fst, vsp.Values[0])
					if err != nil {
						panic(err)
					}
					v.Value = bt.String()
				}
			}

			t.consts = append(t.consts, v)
		}
	}
}

func (t *concreteSyntaxTree) parseVars(specs []ast.Spec) {
	for _, sp := range specs {
		vsp, ok := sp.(*ast.ValueSpec)
		if !ok {
			panic("Var spec is not ValueSpec type, odd, skipping")
		}

		for i, ident := range vsp.Names {
			v := Var{
				Name: ident.Name,
			}

			if vsp.Type != nil {
				v.Type = t.getFieldType(vsp.Type, "")
			}

			if len(vsp.Values) > 0 && i < len(vsp.Values) {
				switch vt := vsp.Values[i].(type) {
				case *ast.BasicLit:
					v.Value = vt.Value
					if vsp.Type == nil {
						// TODO
						// var定义时没有设置类型的，简陋推断
						v.Type = getTypeByValue(vt.Value)
					}
				case *ast.UnaryExpr:
					switch xt := vt.X.(type) {
					case *ast.CompositeLit:
						v.Type = t.getFieldType(xt.Type, "")
					}
				}
				if v.Value == nil {
					fst := token.NewFileSet()
					bt := bytes.NewBufferString("")
					err := format.Node(bt, fst, vsp.Values[0])
					if err != nil {
						panic(err)
					}
					v.Value = bt.String()
				}
			}

			t.vars = append(t.vars, v)
		}
	}
}

func getTypeByValue(v string) Type {
	var t Type
	switch {
	case isInt(v):
		t.Name = "int"
	case isFloat(v):
		t.Name = "float64"
	default:
		t.Name = "string"
	}
	return t
}

func isInt(s string) bool {
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}

	return false
}

func isFloat(s string) bool {
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}

	return false
}

func (t *concreteSyntaxTree) parseFuncDecl(funcDecl *ast.FuncDecl) {
	if funcDecl == nil {
		return
	}

	var method = Method{
		Name: funcDecl.Name.Name,
	}

	if funcDecl.Type != nil {
		method.Recv = t.parseFields(funcDecl.Recv, "")
		method.Params = t.parseFields(funcDecl.Type.Params, "")
		method.Results = t.parseFields(funcDecl.Type.Results, "")

		for _, field := range method.Recv {
			strc, found := t.structMap[t.packageName][field.Type.Name]
			if found {
				strc.Methods = append(strc.Methods, method)
			}
		}
	}

	t.methods = append(t.methods, method)
}

func (t *concreteSyntaxTree) parseImportSpec(specs []ast.Spec) {
	for _, spec := range specs {
		if importSpec, ok := spec.(*ast.ImportSpec); ok {
			var name string
			if importSpec.Name != nil {
				name = importSpec.Name.Name
			}

			t.imports = append(t.imports, Import{
				Alias: name,
				Path:  importSpec.Path.Value,
			})
		}
	}
}

func (t *concreteSyntaxTree) parseTypeSpec(specs []ast.Spec) {
	for _, spec := range specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok {
			typeName := typeSpec.Name.Name
			switch tt := typeSpec.Type.(type) {
			case *ast.InterfaceType:
				t.parseInterface(tt, typeName)
			case *ast.StructType:
				t.addStruct(t.packageName, &Struct{
					Position: t.fset.Position(typeSpec.Name.NamePos),
					Name:     typeName,
					Fields:   t.parseFields(tt.Fields, typeName),
				}, false)
			case *ast.FuncType:
				t.parseFuncType(typeName, tt)
			case *ast.Ident:
				typ := t.getFieldType(tt, typeName)
				t.addStruct(t.packageName, &Struct{
					Position: t.fset.Position(typeSpec.Name.NamePos),
					Name:     typeName,
					Type:     &typ,
				}, true)
			case *ast.SelectorExpr:
				// TODO type Key syscall.Handle
			case *ast.ArrayType:
				// TODO type SpecialCase []CaseRange
			case *ast.StarExpr:
				// TODO type Pointer *ArbitraryType
			case *ast.MapType:
				// TODO type Values map[string][]string
			case *ast.ChanType:
				// TODO type chanWriter chan string
			default:
				panic(fmt.Sprintf("Unknown TypeSpec(type:%T pos:%s) analysis", tt, t.fset.Position(tt.Pos())))
			}
		}
	}
}

func (t *concreteSyntaxTree) addStruct(pkg string, strc *Struct, isTypeDefine bool) {
	if t.structMap[pkg] == nil {
		t.structMap[pkg] = map[string]*Struct{}
	}

	if existsStruct, found := t.structMap[pkg][strc.Name]; found {
		// 如果是类型定义类型的语句，当前存储的structmap中的type为nil
		// 覆盖更新
		if existsStruct.Type == nil && strc.Type != nil && isTypeDefine {
			strc.PackageName = pkg
			t.structMap[pkg][strc.Name] = strc
		}
		return
	}

	// 当前package下所有struct进行存储
	strc.PackageName = pkg
	t.structMap[pkg][strc.Name] = strc
}

func (t *concreteSyntaxTree) parseInterface(it *ast.InterfaceType, iterName string) {
	var iter = Interface{
		Name: iterName,
	}
	if it.Methods == nil {
		return
	}
	iter.Methods = make([]Method, len(it.Methods.List))
	for i, method := range it.Methods.List {
		if len(method.Names) > 0 {
			iter.Methods[i].Name = method.Names[0].Name
		}

		if funcType, ok := method.Type.(*ast.FuncType); ok {
			iter.Methods[i].Params = t.parseFields(funcType.Params, "")
			iter.Methods[i].Results = t.parseFields(funcType.Results, "")
		}
	}
	t.interfaces = append(t.interfaces, iter)
}

func (t *concreteSyntaxTree) parseFuncType(name string, ft *ast.FuncType) {
	t.methods = append(t.methods, Method{
		Name:    name,
		Params:  t.parseFields(ft.Params, ""),
		Results: t.parseFields(ft.Results, ""),
	})
}

func (t *concreteSyntaxTree) parseFields(fieldList *ast.FieldList, structName string) []Field {
	var fields []Field
	if fieldList == nil {
		return fields
	}

	for _, field := range fieldList.List {
		var f = Field{
			Type: t.getFieldType(field.Type, structName),
			Pos:  t.fset.Position(field.Pos()).String(),
		}
		if field.Tag != nil {
			// 去除`防止后续处理出现tag无法解析的问题
			f.Tag = strings.Trim(field.Tag.Value, "`")
		}

		if field.Names != nil {
			// 命名参数
			for _, fieldName := range field.Names {
				// 过滤指定规则的字段
				if t.opts.fieldNameFilter(fieldName.Name) {
					continue
				}

				f.Name = fieldName.Name

				fields = append(fields, f)
			}
		} else {
			// 匿名参数
			fields = append(fields, f)
		}
	}
	return fields
}

func (t *concreteSyntaxTree) getFieldType(expr ast.Expr, structName string) Type {
	var typ Type
	typ.Position = t.fset.Position(expr.Pos())
	switch ex := expr.(type) {
	case *ast.Ident:
		ident := t.getFieldTypeByIdent(ex, t.packageName, structName)
		typ.Name = ident.Name
		typ.GoType = ident.GoType
	case *ast.BasicLit:
		switch ex.Kind {
		case token.INT:
			typ.GoType = BasicType
			typ.Name = "int"
		case token.FLOAT:
			typ.GoType = BasicType
			typ.Name = "float64"
		case token.CHAR:
			typ.GoType = BasicType
			typ.Name = "string"
		case token.STRING:
			typ.GoType = BasicType
			typ.Name = "string"
		case token.IMAG:
			// TODO 类型定义估计要转下 IMAG对应的类型
			typ.GoType = CrossProtocolUnsupportType
		}
	case *ast.SelectorExpr:
		// time.Time
		// X = time
		// Sel = TIme
		if x, ok := ex.X.(*ast.Ident); ok {
			typ.X = x.Name
		}

		typ.Name = ex.Sel.Name
		st := t.getFieldTypeByIdent(ex.Sel, typ.X, structName)
		typ.GoType = st.GoType
	case *ast.StarExpr:
		// *model.XXXStruct
		// X = model
		// Name = XXXStruct
		typ.Star = true
		st := t.getFieldType(ex.X, structName)
		typ.Name = st.Name
		typ.X = st.X
		typ.GoType = st.GoType
	case *ast.InterfaceType:
		typ.Name = "interface{}"
		typ.GoType = CrossProtocolUnsupportType
	case *ast.ArrayType:
		// []model.XXXStruct
		// X = model
		// Name = XXXStruct
		st := t.getFieldType(ex.Elt, structName)
		typ.X = st.X
		typ.Name = "[]" + st.String()
		typ.GoType = ArrayType
		typ.ElementType = &st.BaseType
	case *ast.MapType:
		keyType := t.getFieldType(ex.Key, structName)
		valType := t.getFieldType(ex.Value, structName)
		typ.Name = fmt.Sprintf("map[%s]%s", keyType.String(), valType.String())
		typ.GoType = MapType
		typ.KeyType = &keyType.BaseType
		typ.ValueType = &valType.BaseType
	case *ast.StructType:
		// XXX_NoUnkeyedLiteral struct{} `json:"-"`
		//fmt.Println("----", ex, t.fset.Position(ex.Pos()))
		// TODO
		// typ.GoType = StructType
	case *ast.Ellipsis:
		// (opts ...CallOption)
		typ = t.getFieldType(ex.Elt, "")
		typ.GoType = EllipsisType
	case *ast.FuncType:
		// type XXXFunc func(i int)
		typ.GoType = FuncType
	case *ast.ChanType:
		// var c chan int
		// TODO
	default:
		panic(fmt.Sprintf("Unknown Expr(type:%T pos:%s) analysis", ex, t.fset.Position(ex.Pos())))
	}
	return typ
}

func (t *concreteSyntaxTree) getFieldTypeByIdent(ident *ast.Ident, pkg, structName string) Type {
	var typ Type
	typ.Name = ident.Name
	if IsBasicType(typ.Name) {
		typ.GoType = BasicType
	} else {
		typ.GoType = StructType
		// 防止解析嵌套自身的结构体死循环
		// e.g. type Foo struct{F Foo} 自己嵌套了自己
		if structName != typ.Name {
			t.parseStruct(ident, pkg, structName)
		}
	}
	return typ
}

func IsBasicType(typeName string) bool {
	if obj := types.Universe.Lookup(typeName); obj != nil {
		return true
	}
	return false
}

func (t *concreteSyntaxTree) parseStruct(id *ast.Ident, X, structName string) {
	// X 是字段的引用名 e.g. model.User这里的model
	if X != "" && X != t.packageName {
		// 如果入口处的structName是 XXXRequest或者XXXResponse之类后缀的结构体
		// 尝试解析嵌套的其他包中的struct,例如model.Foo,取model文件夹中搜索
		if strings.HasSuffix(structName, utils.GetRequestSuffix()) ||
			strings.HasSuffix(structName, utils.GetResponseSuffix()) {
			t.parseReferencePackage(X)
		}
		return
	}

	pkg := t.packageName
	s := &Struct{
		Name:        id.Name,
		Position:    t.fset.Position(id.NamePos),
		PackageName: pkg,
	}

	if id.Obj != nil && id.Obj.Decl != nil {
		s.Position = t.fset.Position(id.Obj.Pos())
		if typeSpec, ok := id.Obj.Decl.(*ast.TypeSpec); ok {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				s.Fields = t.parseFields(structType.Fields, id.Name)
			}
		}
	}

	if _, found := t.structMap[t.packageName][s.Name]; found {
		return
	}

	t.addStruct(pkg, s, false)
}

func (t *concreteSyntaxTree) parseReferencePackage(pkg string) {
	if pkg == "" {
		return
	}

	for _, imp := range t.imports {
		impPkg := imp.Alias
		// 没有取别名的情况下这里是空字符串
		if impPkg == "" {
			impPkg = strings.Trim(path.Base(imp.Path), "\"")
		}

		// 引用包解析过滤，否则会发生堆栈溢出
		if impPkg != pkg {
			continue
		}

		if _, found := t.parsedReferencePackageMap[imp.Path]; !found {
			t.parsedReferencePackageMap[imp.Path] = struct{}{}
		} else {
			return
		}

		if _, found := t.structMap[pkg]; found {
			return
		}

		for _, path := range []string{"GOPATH", "GOROOT"} {
			gopath := os.Getenv(path)
			filePath := filepath.Join(gopath, "src", strings.Trim(imp.Path, "\""))
			fileinfos, err := ioutil.ReadDir(filePath)
			if err != nil {
				// TODO 文件无法找到是否需要处理
				continue
			}
			for _, fileinfo := range fileinfos {
				if fileinfo.IsDir() || !strings.HasSuffix(fileinfo.Name(), ".go") {
					continue
				}
				fset := token.NewFileSet()
				f, err := parser.ParseFile(fset, filepath.Join(filePath, fileinfo.Name()), nil, 0)
				if err != nil {
					panic(err)
				}

				t2 := newConcreteSyntaxTree(fset, f)
				t2.parsedReferencePackageMap = t.parsedReferencePackageMap
				err = t2.Parse()
				if err != nil {
					panic(err)
				}
				t.mergeStructMap(t2)
			}
		}
	}
}

func (t *concreteSyntaxTree) mergeStructMap(t2 *concreteSyntaxTree) *concreteSyntaxTree {

	// 将a2建立的结构集合合并到主的AST StructMap中
	if t.structMap[t2.packageName] == nil {
		t.structMap[t2.packageName] = t2.structMap[t2.packageName]
	}
	// 合并解析过的包集合
	for key, val := range t2.parsedReferencePackageMap {
		if _, found := t.parsedReferencePackageMap[key]; !found {
			t.parsedReferencePackageMap[key] = val
		}
	}
	return t
}
