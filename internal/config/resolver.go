package config

import "reflect"

type Resolver interface {
	Key() string
	Priority() int
	Resolve(path []string, valueType reflect.Type) (any, bool, error)
}

type baseResolver struct {
	key      string
	priority int
}

func (r *baseResolver) Key() string {
	return r.key
}

func (r *baseResolver) Priority() int {
	return r.priority
}

type ResolverOption struct {
	apply func(*baseResolver)
}

func WithKey(key string) ResolverOption {
	return ResolverOption{
		apply: func(r *baseResolver) {
			r.key = key
		},
	}
}

func WithPriority(priority int) ResolverOption {
	return ResolverOption{
		apply: func(r *baseResolver) {
			r.priority = priority
		},
	}
}

func (r *baseResolver) applyOptions(options ...ResolverOption) {
	for _, o := range options {
		o.apply(r)
	}
}
