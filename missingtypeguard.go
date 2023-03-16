package missingtypeguard

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "missingtypeguard is ..."

// Analyzer is ...
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

	typeGuardOwnersByInterfaces := make(map[types.Type]map[string]struct{})

	// find interfaces in the package
	inspect.Preorder([]ast.Node{(*ast.TypeSpec)(nil)}, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.TypeSpec:
			if _, ok := n.Type.(*ast.InterfaceType); !ok {
				return
			}

			itype := pass.TypesInfo.TypeOf(n.Name)
			typeGuardOwnersByInterfaces[itype] = make(map[string]struct{})
		}
	})

	// find all type guards for `Animal`
	inspect.Preorder([]ast.Node{(*ast.ValueSpec)(nil)}, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.ValueSpec:
			if n.Type == nil || len(n.Values) != 1 {
				return
			}

			itype := pass.TypesInfo.TypeOf(n.Type)
			ntype := pass.TypesInfo.TypeOf(n.Values[0])
			typeGuardOwnersByInterfaces[itype][ntype.String()] = struct{}{}
		}
	})

	// find structs missing type guards for `Animal`
	inspect.Preorder([]ast.Node{(*ast.TypeSpec)(nil)}, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.TypeSpec:
			if _, ok := n.Type.(*ast.StructType); !ok {
				return
			}

			i, ok := pass.Pkg.Scope().Lookup("Animal").Type().Underlying().(*types.Interface)
			if !ok {
				return
			}

			for _, typeGuardOwners := range typeGuardOwnersByInterfaces {
				ntype := pass.TypesInfo.TypeOf(n.Name)
				if types.Implements(ntype, i) {
					if _, ok := typeGuardOwners[ntype.String()]; !ok {
						pass.Reportf(n.Pos(), "%s is missing a type guard for Animal", n.Name.Name)
					}

					return // no need to check for pointer
				}

				nptype := types.NewPointer(ntype)
				if types.Implements(nptype, i) {
					if _, ok := typeGuardOwners[nptype.String()]; !ok {
						pass.Reportf(n.Pos(), "%s is missing a type guard for Animal", n.Name.Name)
					}
				}
			}
		}
	})

	return nil, nil
}
