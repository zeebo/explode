package main

import (
	"go/types"
	"reflect"
	"sort"
)

type objectStack []types.Object

func (s *objectStack) Push(obj types.Object) {
	*s = append(*s, obj)
}

func (s *objectStack) Pop() types.Object {
	val := s.Top()
	if val == nil {
		return nil
	}
	*s = (*s)[:len(*s)-1]
	return val
}

func (s *objectStack) Top() types.Object {
	if len(*s) == 0 {
		return nil
	}
	return (*s)[len(*s)-1]
}

type byPtr []types.Object

func (b byPtr) Len() int { return len(b) }

func (b byPtr) ptr(i int) uintptr {
	return reflect.ValueOf(&b[i]).Elem().InterfaceData()[1]
}

func (b byPtr) Less(i int, j int) bool {
	return b.ptr(i) < b.ptr(j)
}

func (b byPtr) Swap(i int, j int) {
	b[i], b[j] = b[j], b[i]
}

type objectGraph map[types.Object]objectSet

func (o objectGraph) keys() (out []types.Object) {
	for obj := range o {
		out = append(out, obj)
	}
	sort.Sort(byPtr(out))
	return out
}

func (o objectGraph) depthFirst() (out []types.Object) {
	var set objectSet
	for _, key := range o.keys() {
		out = append(out, o.depthFirstFrom(key, &set)...)
	}
	return out
}

func (o objectGraph) depthFirstFrom(key types.Object, set *objectSet) (
	out []types.Object) {

	if set.Has(key) {
		return
	}
	set.Add(key)

	for target := range o[key] {
		out = append(out, o.depthFirstFrom(target, set)...)
	}

	out = append(out, key)
	return out
}

func (o objectGraph) Transpose() (out objectGraph) {
	out = objectGraph{}
	for key, set := range o {
		for target := range set {
			target_set := out[target]
			target_set.Add(key)
			out[target] = target_set
		}
	}
	return out
}

func scc(graph objectGraph) (sccs []objectSet) {
	var stack objectStack
	for _, key := range graph.depthFirst() {
		stack.Push(key)
	}

	transpose := graph.Transpose()

	var visited objectSet
	for {
		top := stack.Pop()
		if top == nil {
			break
		}

		if visited.Has(top) {
			continue
		}

		var component objectSet
		for _, key := range transpose.depthFirstFrom(top, &visited) {
			component.Add(key)
		}
		sccs = append(sccs, component)
	}

	return sccs
}
