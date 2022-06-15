package tslist

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"sync"

	"github.com/samber/lo"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const doc = "tslist is ..."

const INF = 1 << 60

const (
	ANY   = "any"
	TILDA = "~"
)

type Visitor struct {
	nest          int
	pass          *analysis.Pass
	interfaceName string
	methodNames   []string
	methodResults []Method
	mp            map[string]types.Type
	mu            sync.Mutex
	typeResults   map[int][]string
}

type VisitorResult struct {
	Pos      token.Pos
	Name     string
	TypeSets []TypeValue
	Methods  []Method
}

type Method struct {
	Name    string
	Args    []TypeValue
	Outputs []TypeValue
}

type TypeValue struct {
	Name string
	Type types.Type
}

type Result struct {
	Results []VisitorResult
}

// Analyzer is ...
var Analyzer = &analysis.Analyzer{
	Name: "tslist",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	ResultType: reflect.TypeOf(new(Result)),
}

func run(pass *analysis.Pass) (interface{}, error) {
	var result Result
	for _, f := range pass.Files {
		for _, dec := range f.Decls {
			dec, _ := dec.(*ast.GenDecl)
			if dec == nil || dec.Tok != token.TYPE {
				continue
			}

			for _, spec := range dec.Specs {
				spec, _ := spec.(*ast.TypeSpec)
				if spec == nil || spec.Type == nil {
					continue
				}

				interfaceType, _ := spec.Type.(*ast.InterfaceType)
				if interfaceType == nil {
					continue
				}

				res := InterfaceVisitor(spec.Name.Name, interfaceType, pass)
				result.Results = append(result.Results, res)
			}
		}
	}

	for _, res := range result.Results {
		ts := typeSetPrint(res)
		methodList := methodListPrint(res)
		format := fmt.Sprintf("\n%s\ntype set: %v\nmethod list:\n", res.Name, ts)
		for _, method := range methodList {
			format += fmt.Sprintf("%s\n", method)
		}
		pass.Reportf(res.Pos, format)
	}

	return &result, nil
}

func InterfaceVisitor(name string, interfaceType *ast.InterfaceType, pass *analysis.Pass) VisitorResult {
	visit := Visitor{pass: pass, interfaceName: name, typeResults: make(map[int][]string), mp: make(map[string]types.Type)}
	visit.interfaceVisitor(interfaceType)

	typeSet := visit.parseTypeSet()
	visit.parseMethodList()

	return VisitorResult{Pos: interfaceType.Pos(), Name: name, TypeSets: typeSet, Methods: visit.methodResults}
}

func (v *Visitor) parseTypeSet() []TypeValue {
	typeSet := make(map[string]int)
	// union
	for _, results := range v.typeResults {
		if lo.Contains(results, ANY) {
			typeSet[ANY]++
			continue
		}

		for _, result := range results {
			typeSet[result]++
		}
	}

	res := make([]TypeValue, 0, len(typeSet))
	// intersection
	if _, ok := typeSet[ANY]; ok {
		if len(typeSet) == 1 {
			if typ, ok := v.mp[ANY]; ok {
				res = append(res, TypeValue{ANY, typ})
				return res
			}
			res = append(res, TypeValue{Name: ANY})
			return res
		}

		v.nest -= typeSet[ANY]
		typeSet[ANY] = INF
	}

	for typ := range typeSet {
		if strings.HasPrefix(typ, TILDA) {
			defaultType := strings.Trim(typ, TILDA)
			if _, ok := typeSet[defaultType]; ok {
				typeSet[defaultType]++
			}
		}
	}

	for typeName, num := range typeSet {
		if num == v.nest {
			if typ, ok := v.mp[typeName]; ok {
				res = append(res, TypeValue{typeName, typ})
			}
		}
	}

	return res
}

func (v *Visitor) parseMethodList() {
	for i, name := range v.methodNames {
		v.methodResults[i].Name = name
	}
}

func (v *Visitor) interfaceVisitor(expr *ast.InterfaceType) {
	if expr.Methods == nil {
		return
	}

	if expr.Methods.List == nil {
		v.nest++
		v.typeResults[v.nest] = append(v.typeResults[v.nest], ANY)
	}

	for _, field := range expr.Methods.List {
		v.nest++
		for _, name := range field.Names {
			v.methodNames = append(v.methodNames, name.Name)
		}
		v.exprVisitor(field.Type)
	}
}

func (v *Visitor) exprVisitor(expr ast.Expr) {
	switch expr := expr.(type) {
	case *ast.BinaryExpr:
		v.exprVisitor(expr.X)
		v.exprVisitor(expr.Y)
	case *ast.Ident:
		v.identVisitor(expr)
	case *ast.UnaryExpr:
		v.unaryVisitor(expr)
	case *ast.FuncType:
		v.funcTypeVisitor(expr)
	case *ast.StarExpr:
		v.starExprVisitor(expr)
	case *ast.ArrayType:
		v.arrayTypeVisitor(expr)
	}
}

func (v *Visitor) identVisitor(expr *ast.Ident) {
	if expr.Obj == nil || expr.Obj.Decl == nil {
		typ := v.pass.TypesInfo.TypeOf(expr)
		v.addType(typ.String(), typ)
		v.typeResults[v.nest] = append(v.typeResults[v.nest], typ.String())
	} else {
		dec, _ := expr.Obj.Decl.(*ast.TypeSpec)
		if dec == nil || dec.Type == nil {
			return
		}
		switch dec := dec.Type.(type) {
		case *ast.InterfaceType:
			res := InterfaceVisitor(v.interfaceName, dec, v.pass)
			for _, ts := range res.TypeSets {
				v.typeResults[v.nest] = append(v.typeResults[v.nest], ts.Name)
			}
		case *ast.Ident:
			typ := v.pass.TypesInfo.TypeOf(dec)
			v.addType(expr.Name, typ)
			v.typeResults[v.nest] = append(v.typeResults[v.nest], expr.Name)
		case *ast.StructType:
			typ := v.pass.TypesInfo.TypeOf(dec)
			name := fmt.Sprintf("%s.%s", v.pass.Pkg.Name(), expr.Name)
			v.addType(name, typ)
			v.typeResults[v.nest] = append(v.typeResults[v.nest], name)
		}
	}
}

func (v *Visitor) unaryVisitor(expr *ast.UnaryExpr) {
	if expr.Op != token.TILDE {
		return
	}

	exprX, _ := expr.X.(*ast.Ident)
	if exprX == nil {
		return
	}

	typ := v.pass.TypesInfo.TypeOf(exprX)
	name := fmt.Sprintf("~%s", typ.String())
	v.addType(name, typ)
	v.typeResults[v.nest] = append(v.typeResults[v.nest], name)
}

func (v *Visitor) funcTypeVisitor(expr *ast.FuncType) {
	v.nest--
	var method Method
	if expr.Params != nil && expr.Params.List != nil {
		values := v.params(expr.Params.List)
		method.Args = values
	}

	if expr.Results != nil && expr.Results.List != nil {
		values := v.params(expr.Results.List)
		method.Outputs = values
	}

	v.methodResults = append(v.methodResults, method)
}

func (v *Visitor) starExprVisitor(expr *ast.StarExpr) {
	typ := v.pass.TypesInfo.TypeOf(expr.X)
	name := fmt.Sprintf("*%s", typ.String())
	v.addType(name, typ)
	v.typeResults[v.nest] = append(v.typeResults[v.nest], name)
}

func (v *Visitor) arrayTypeVisitor(expr *ast.ArrayType) {
	typ := v.pass.TypesInfo.TypeOf(expr.Elt)
	name := fmt.Sprintf("[]%s", typ.String())
	v.addType(name, typ)
	v.typeResults[v.nest] = append(v.typeResults[v.nest], name)
}

func (v *Visitor) params(fields []*ast.Field) []TypeValue {
	values := make([]TypeValue, 0, len(fields))
	for _, field := range fields {
		if field.Names == nil {
			typ := v.pass.TypesInfo.TypeOf(field.Type)
			values = append(values, TypeValue{"", typ})
			continue
		}

		for _, fieldName := range field.Names {
			typ := v.pass.TypesInfo.TypeOf(fieldName)
			values = append(values, TypeValue{fieldName.Name, typ})
		}
	}

	return values
}

func (v *Visitor) addType(name string, typ types.Type) {
	v.mu.Lock()
	if _, ok := v.mp[name]; !ok {
		v.mp[name] = typ
	}
	v.mu.Unlock()
}
