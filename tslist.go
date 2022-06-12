package tslist

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"github.com/samber/lo"
	"golang.org/x/exp/typeparams"
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
	methodResults []MethodResult
	typeResults   map[int][]string
}

type MethodResult struct {
	inputs  []value
	outputs []value
}

type value struct {
	name     string
	typeName string
}

func (v value) isNoName() bool {
	if v.name == "" {
		return true
	}

	return false
}

type VisitorResult struct {
	Pos           token.Pos
	Name          string
	MethodResults map[string]MethodResult
	TypeResults   []string
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
}

func run(pass *analysis.Pass) (interface{}, error) {
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

				typ := pass.TypesInfo.TypeOf(interfaceType)
				terms, err := typeparams.NormalTerms(typ)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(terms)
				}
				res := InterfaceVisitor(spec.Name.Name, interfaceType, pass)
				for name, values := range res.MethodResults {
					format := name
					if len(values.inputs) == 0 {
						format += " ()"
					} else {
						format = addValues(format, values.inputs)
					}
					if len(values.outputs) == 1 {
						if values.outputs[0].isNoName() {
							format += fmt.Sprintf(" %s", values.outputs[0].typeName)
						} else {
							format += fmt.Sprintf(" (%s %s)", values.outputs[0].name, values.outputs[0].typeName)
						}
					} else {
						format = addValues(format, values.outputs)
					}
					fmt.Println(format)
				}
				if len(res.TypeResults) == 0 {
					pass.Reportf(res.Pos, "no type")
					fmt.Printf("%s: no type set\n", res.Name)
				} else {
					sort.Slice(res.TypeResults, func(i, j int) bool { return res.TypeResults[i] < res.TypeResults[j] })
					pass.Reportf(res.Pos, "%v", res.TypeResults)
					fmt.Printf("%s: %v\n", res.Name, res.TypeResults)
				}
			}
		}
	}
	return nil, nil
}

func addValues(format string, values []value) string {
	format += " ("
	for i, value := range values {
		if i == len(values)-1 {
			format = addValue(format, value)
		} else {
			format = addValue(format, value)
			format += ", "
		}
	}

	format += ")"
	return format
}

func addValue(format string, value value) string {
	if value.isNoName() {
		format += value.typeName
		return format
	}
	format += fmt.Sprintf("%s %s", value.name, value.typeName)
	return format
}

func InterfaceVisitor(name string, interfaceType *ast.InterfaceType, pass *analysis.Pass) VisitorResult {
	mp := make(map[int][]string)
	visit := Visitor{pass: pass, interfaceName: name, typeResults: mp}
	visit.interfaceVisitor(interfaceType)

	res := visit.parseTypeSet()
	methodMap := visit.parseMethodList()

	return VisitorResult{interfaceType.Pos(), name, methodMap, res}
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

func (v *Visitor) parseMethodList() map[string]MethodResult {
	methodMap := make(map[string]MethodResult)
	for i, name := range v.methodNames {
		methodMap[name] = v.methodResults[i]
	}

	return methodMap
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
	}
}

func (v *Visitor) identVisitor(expr *ast.Ident) {
	if expr.Obj == nil || expr.Obj.Decl == nil {
		typ := v.pass.TypesInfo.TypeOf(expr)
		v.typeResults[v.nest] = append(v.typeResults[v.nest], typ.String())
	} else {
		dec, _ := expr.Obj.Decl.(*ast.TypeSpec)
		if dec == nil || dec.Type == nil {
			return
		}
		switch dec := dec.Type.(type) {
		case *ast.InterfaceType:
			res := InterfaceVisitor(v.interfaceName, dec, v.pass)
			for _, typ := range res.TypeResults {
				v.typeResults[v.nest] = append(v.typeResults[v.nest], typ)
			}
		case *ast.Ident:
			typ := v.pass.TypesInfo.TypeOf(dec)
			v.typeResults[v.nest] = append(v.typeResults[v.nest], typ.String())
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
	v.typeResults[v.nest] = append(v.typeResults[v.nest], fmt.Sprintf("~%s", typ.String()))
}

func (v *Visitor) funcTypeVisitor(expr *ast.FuncType) {
	v.nest--
	var methodResult MethodResult
	if expr.Params != nil && expr.Params.List != nil {
		values := v.params(expr.Params.List)
		methodResult.inputs = values
	}

	if expr.Results != nil && expr.Results.List != nil {
		values := v.params(expr.Results.List)
		methodResult.outputs = values
	}

	v.methodResults = append(v.methodResults, methodResult)
}

func (v *Visitor) params(fields []*ast.Field) []value {
	values := make([]value, 0, len(fields))
	for _, field := range fields {
		if field.Names == nil {
			typ := v.pass.TypesInfo.TypeOf(field.Type)
			values = append(values, value{"", typ.String()})
			continue
		}

		for _, fieldName := range field.Names {
			typ := v.pass.TypesInfo.TypeOf(fieldName)
			values = append(values, value{fieldName.Name, typ.String()})
		}
	}

	return values
}
