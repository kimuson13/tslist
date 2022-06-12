package tslist

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

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
	typeResults   map[int][]string
}

type VisitorResult struct {
	Pos     token.Pos
	Name    string
	TypeSet []TypeValue
	Methods []Method
}

type Method struct {
	Name    string
	Args    []TypeValue
	Outputs []TypeValue
}

type TypeValue struct {
	Name string
	Typ  types.Type
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

var mp = make(map[string]types.Type)

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
				fmt.Println(res)
			}
		}
	}

	return &result, nil
}

func InterfaceVisitor(name string, interfaceType *ast.InterfaceType, pass *analysis.Pass) VisitorResult {
	visit := Visitor{pass: pass, interfaceName: name, typeResults: make(map[int][]string)}
	visit.interfaceVisitor(interfaceType)

	res := visit.parseTypeSet()
	visit.parseMethodList()

	// set result process
	typeSet := make([]TypeValue, 0, len(res))
	for _, name := range res {
		if typ, ok := mp[name]; ok {
			typeSet = append(typeSet, TypeValue{name, typ})
		}
	}

	return VisitorResult{Pos: interfaceType.Pos(), Name: name, TypeSet: typeSet, Methods: visit.methodResults}
}

func (v *Visitor) parseTypeSet() []string {
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

	res := make([]string, 0, len(typeSet))
	// intersection
	if _, ok := typeSet[ANY]; ok {
		if len(typeSet) == 1 {
			res = append(res, ANY)
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

	for typ, num := range typeSet {
		if num == v.nest {
			res = append(res, typ)
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
		typ := v.pass.TypesInfo.TypeOf(expr.X)
		name := fmt.Sprintf("*%s", typ.String())
		addType(name, typ)
		v.typeResults[v.nest] = append(v.typeResults[v.nest], name)
	case *ast.ArrayType:
		typ := v.pass.TypesInfo.TypeOf(expr.Elt)
		name := fmt.Sprintf("[]%s", typ.String())
		addType(name, typ)
		v.typeResults[v.nest] = append(v.typeResults[v.nest], name)
	}
}

func (v *Visitor) identVisitor(expr *ast.Ident) {
	if expr.Obj == nil || expr.Obj.Decl == nil {
		typ := v.pass.TypesInfo.TypeOf(expr)
		addType(typ.String(), typ)
		v.typeResults[v.nest] = append(v.typeResults[v.nest], typ.String())
	} else {
		dec, _ := expr.Obj.Decl.(*ast.TypeSpec)
		if dec == nil || dec.Type == nil {
			return
		}
		switch dec := dec.Type.(type) {
		case *ast.InterfaceType:
			res := InterfaceVisitor(v.interfaceName, dec, v.pass)
			for _, ts := range res.TypeSet {
				v.typeResults[v.nest] = append(v.typeResults[v.nest], ts.Name)
			}
		case *ast.Ident:
			typ := v.pass.TypesInfo.TypeOf(dec)
			addType(typ.String(), typ)
			v.typeResults[v.nest] = append(v.typeResults[v.nest], typ.String())
		case *ast.StructType:
			typ := v.pass.TypesInfo.TypeOf(dec)
			name := fmt.Sprintf("%s.%s", v.pass.Pkg.Name(), expr.Name)
			addType(name, typ)
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
	addType(name, typ)
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

func (v *Visitor) params(fields []*ast.Field) []TypeValue {
	values := make([]TypeValue, 0, len(fields))
	for _, field := range fields {
		if field.Names == nil {
			typ := v.pass.TypesInfo.TypeOf(field.Type)
			values = append(values, TypeValue{typ.String(), typ})
			continue
		}

		for _, fieldName := range field.Names {
			typ := v.pass.TypesInfo.TypeOf(fieldName)
			values = append(values, TypeValue{typ.String(), typ})
		}
	}

	return values
}

func addType(name string, typ types.Type) {
	if _, ok := mp[name]; !ok {
		mp[name] = typ
	}
}
