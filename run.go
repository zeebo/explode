package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types/typeutil"
)

func run(ctx context.Context, input, output string) error {
	var conf loader.Config
	conf.Import(input)

	prog, err := conf.Load()
	if err != nil {
		return wrap(err)
	}

	pkg, ok := prog.Imported[input]
	if !ok {
		fatal("unable to find imported %q", input)
	}

	r := &resolver{
		prog: prog,
		pkg:  pkg,
	}
	r.init()

	graph := r.computeGraph()
	sccs := scc(graph)

	for _, scc := range sccs {
		fmt.Println("scc ====================")
		for key := range scc {
			fmt.Println("\t", key)
		}
	}

	return nil
}

type resolver struct {
	prog     *loader.Program
	pkg      *loader.PackageInfo
	toplevel objectSet
	types    typeutil.Map // map from types.Type => types.Object of decl
}

func (r *resolver) init() {
	scope := r.pkg.Pkg.Scope()
	for _, name := range scope.Names() {
		switch obj := scope.Lookup(name).(type) {
		case *types.Func, *types.Var, *types.Const:
			r.toplevel.Add(obj)
		case *types.TypeName:
			r.toplevel.Add(obj)
			r.types.Set(obj.Type(), obj)
		}
	}
}

func (r *resolver) findType(typ types.Type) types.Object {
	obj, _ := r.types.At(typ).(types.Object)
	return obj
}

func (r *resolver) computeGraph() objectGraph {
	refs := objectGraph{}

	for obj := range r.toplevel {
		set := objectSet{}
		r.loadReferrers(obj, set)
		refs[obj] = set
	}

	// walk method sets of all the types
	for obj := range r.toplevel {
		typ, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}

		methods := types.NewMethodSet(types.NewPointer(typ.Type()))
		for i := 0; i < methods.Len(); i++ {
			method := methods.At(i).Obj().(*types.Func)
			if method.Pkg() != r.pkg.Pkg {
				continue
			}
			set := objectSet{}
			r.loadReferrers(method, set)
			refs[method] = set
		}
	}

	return refs
}

// given an object, we return every object that it refers to.
func (r *resolver) loadReferrers(obj types.Object, set objectSet) {
	switch o := obj.(type) {
	case *types.TypeName:
		r.loadTypeReferrers(obj, set)
		methods := types.NewMethodSet(types.NewPointer(o.Type()))
		for i := 0; i < methods.Len(); i++ {
			method := methods.At(i).Obj().(*types.Func)
			if method.Pkg() != r.pkg.Pkg {
				continue
			}
			set.Add(method)
		}

	case *types.Func:
		r.loadTypeReferrers(obj, set)
		_, path, exact := r.prog.PathEnclosingInterval(o.Pos(), o.Pos())
		if !exact {
			fatal("inexact enclosing interval")
		}
		r.loadNode(obj, path[1], set)

	case *types.Var:
		_, path, exact := r.prog.PathEnclosingInterval(o.Pos(), o.Pos())
		if !exact {
			fatal("inexact enclosing interval")
		}
		r.loadNode(obj, path[1], set)

	case *types.Const:
		_, path, exact := r.prog.PathEnclosingInterval(o.Pos(), o.Pos())
		if !exact {
			fatal("inexact enclosing interval")
		}
		r.loadNode(obj, path[1], set)

	default:
		fatal("unknown object type: %T", o)
	}
}

type visitFunc func(ast.Node)

func (v visitFunc) Visit(node ast.Node) ast.Visitor { v(node); return v }

func (r *resolver) loadNode(obj types.Object, node ast.Node, set objectSet) {
	v := func(node ast.Node) {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return
		}

		cand := r.pkg.Uses[ident]
		if cand != obj && r.toplevel.Has(cand) {
			set.Add(cand)
		}
	}

	ast.Walk(visitFunc(v), node)
}

func (r *resolver) loadTypeReferrers(obj types.Object, set objectSet) {
	var seen typeutil.Map

	var loadType func(types.Type)
	loadType = func(typ types.Type) {
		if seen.At(typ) != nil {
			return
		}
		seen.Set(typ, true)

		found := r.findType(typ)
		if found != nil && found != obj {
			// once we add an object for a type, we don't have to go any deeper
			// because that object will point to its referrers for us.
			set.Add(found)
			return
		}

		switch t := typ.(type) {
		case *types.Named:
			loadType(t.Underlying())

		case *types.Signature:
			params := t.Params()
			for i := 0; i < params.Len(); i++ {
				loadType(params.At(i).Type())
			}
			results := t.Results()
			for i := 0; i < results.Len(); i++ {
				loadType(results.At(i).Type())
			}
			if recv := t.Recv(); recv != nil {
				loadType(recv.Type())
			}

		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				loadType(t.Field(i).Type())
			}

		case *types.Interface:
			t = t.Complete()
			for i := 0; i < t.NumMethods(); i++ {
				loadType(t.Method(i).Type())
			}

		case *types.Chan:
			loadType(t.Elem())
		case *types.Slice:
			loadType(t.Elem())
		case *types.Array:
			loadType(t.Elem())
		case *types.Map:
			loadType(t.Key())
			loadType(t.Elem())
		case *types.Pointer:
			loadType(t.Elem())

		case *types.Basic:

		default:
			fatal("unknown type: %T", t)
		}
	}

	loadType(obj.Type())
}
