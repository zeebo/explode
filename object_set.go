package main

import "go/types"

type objectSet map[types.Object]struct{}

func (s *objectSet) Add(obj types.Object) {
	if obj == nil {
		return
	}
	if *s == nil {
		*s = objectSet{}
	}
	(*s)[obj] = struct{}{}
}

func (s *objectSet) Delete(obj types.Object) {
	if obj == nil {
		return
	}
	if *s == nil {
		return
	}
	delete(*s, obj)
}

func (s objectSet) Has(obj types.Object) bool {
	if obj == nil {
		return false
	}
	_, ok := s[obj]
	return ok
}

func (s *objectSet) Union(set objectSet) {
	for obj := range set {
		s.Add(obj)
	}
}

func (s *objectSet) Filter(by objectSet) {
	if *s == nil {
		return
	}
	for obj := range *s {
		if !by.Has(obj) {
			s.Delete(obj)
		}
	}
}
