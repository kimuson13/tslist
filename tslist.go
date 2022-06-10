package tslist

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const doc = "tslist is ..."

type Visitor struct {
	nest   int
	pass   *analysis.Pass
	name   string
	result map[int][]string
}

type VisitorResult struct {
	Pos    token.Pos
	Name   string
	Result []string
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

				res := InterfaceVisitor(spec.Name.Name, interfaceType, pass)
				if len(res.Result) == 0 {
					pass.Reportf(res.Pos, "no type")
					// fmt.Printf("%s: no type set\n", res.Name)
				} else {
					sort.Slice(res.Result, func(i, j int) bool { return res.Result[i] < res.Result[j] })
					pass.Reportf(res.Pos, "%v", res.Result)
					// fmt.Printf("%s: %v\n", res.Name, res.Result)
				}
			}
		}
	}

	return nil, nil
}

func InterfaceVisitor(name string, interfaceType *ast.InterfaceType, pass *analysis.Pass) VisitorResult {
	mp := make(map[int][]string)
	visit := Visitor{pass: pass, name: name, result: mp}
	visit.interfaceVisitor(interfaceType)

	typeSet := make(map[string]int)
	for _, results := range visit.result {
		if first := results[0]; len(results) == 1 && first == "any" {
			typeSet[first]++
			continue
		}
		for _, result := range results {
			if result != "any" {
				typeSet[result]++
			}
		}
	}

	res := make([]string, 0, len(typeSet))
	if _, ok := typeSet["any"]; ok {
		if len(typeSet) == 1 {
			res = append(res, "any")
			return VisitorResult{interfaceType.Pos(), name, res}
		}

		visit.nest -= typeSet["any"]
		typeSet["any"]++
	}

	for typ := range typeSet {
		if strings.HasPrefix(typ, "~") {
			val := strings.Trim(typ, "~")
			if _, ok := typeSet[val]; ok {
				typeSet[val]++
			}
		}
	}

	for typ, num := range typeSet {
		if num == visit.nest {
			res = append(res, typ)
		}
	}

	return VisitorResult{interfaceType.Pos(), name, res}
}

func (v *Visitor) interfaceVisitor(expr *ast.InterfaceType) {
	if expr.Methods == nil {
		return
	}

	if expr.Methods.List == nil {
		v.nest++
		v.result[v.nest] = append(v.result[v.nest], "any")
	}

	for _, field := range expr.Methods.List {
		v.nest++
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
		v.nest--
	}
}

func (v *Visitor) identVisitor(expr *ast.Ident) {
	if expr.Obj == nil || expr.Obj.Decl == nil {
		typ := v.pass.TypesInfo.TypeOf(expr)
		v.result[v.nest] = append(v.result[v.nest], typ.String())
	} else {
		dec, _ := expr.Obj.Decl.(*ast.TypeSpec)
		if dec == nil || dec.Type == nil {
			return
		}
		switch dec := dec.Type.(type) {
		case *ast.InterfaceType:
			res := InterfaceVisitor(v.name, dec, v.pass)
			for _, typ := range res.Result {
				v.result[v.nest] = append(v.result[v.nest], typ)
			}
		case *ast.Ident:
			typ := v.pass.TypesInfo.TypeOf(dec)
			v.result[v.nest] = append(v.result[v.nest], typ.String())
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
	v.result[v.nest] = append(v.result[v.nest], fmt.Sprintf("~%s", typ.String()))
}
