package missingtypeguard

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "missingtypeguard checks if types that implement an interface have a type guard for it"

var Analyzer = &analysis.Analyzer{
	Name: "missingtypeguard",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	typeGuardOwnersByInterfaces := typedMap[*typedMap[bool]]{}

	// find interfaces in the package
	inspect.Preorder([]ast.Node{(*ast.TypeSpec)(nil)}, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.TypeSpec:
			if _, ok := n.Type.(*ast.InterfaceType); !ok {
				return
			}

			itype := pass.TypesInfo.TypeOf(n.Name)
			typeGuardOwnersByInterfaces.Set(itype, &typedMap[bool]{})
		}
	})

	// find all type guards
	inspect.Preorder([]ast.Node{(*ast.ValueSpec)(nil)}, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.ValueSpec:
			if n.Type == nil || len(n.Values) != 1 {
				return
			}

			itype := pass.TypesInfo.TypeOf(n.Type)
			ntype := pass.TypesInfo.TypeOf(n.Values[0])

			if typeGuardOwnersByInterfaces.At(itype) == nil {
				fmt.Println("warning: multi package is not completely supported yet")
				typeGuardOwnersByInterfaces.Set(itype, &typedMap[bool]{})
			}

			typeGuardOwnersByInterfaces.At(itype).Set(ntype, true)
		}
	})

	// find structs missing type guards
	inspect.Preorder([]ast.Node{(*ast.TypeSpec)(nil)}, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.TypeSpec:
			if _, ok := n.Type.(*ast.InterfaceType); ok {
				return
			}

			typeGuardOwnersByInterfaces.Iterate(func(itype types.Type, typeGuardOwners *typedMap[bool]) {
				i, ok := itype.Underlying().(*types.Interface)
				if !ok {
					return
				}

				ntype := pass.TypesInfo.TypeOf(n.Name)
				if types.Implements(ntype, i) {
					if ok := typeGuardOwners.At(ntype); !ok {
						pass.Reportf(n.Pos(), "%s is missing a type guard for %s", ntype.String(), itype.String())
					}

					return // no need to check for pointer
				}

				nptype := types.NewPointer(ntype)
				if types.Implements(nptype, i) {
					if ok := typeGuardOwners.At(nptype); !ok {
						pass.Reportf(n.Pos(), "the pointer of %s is missing a type guard for %s", ntype.String(), itype.String())
					}
				}
			})
		}
	})

	return nil, nil
}
