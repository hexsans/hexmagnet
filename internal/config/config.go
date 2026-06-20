package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/iancoleman/strcase"
)

type SpecEntry struct {
	Key           string
	DefaultValue  any
	ValidatorOpts []ValidatorOption
}

type Spec struct {
	Key          string
	DefaultValue any
}

type ResolvedConfig struct {
	NodeMap map[string]ResolvedNode
}

func (r ResolvedConfig) Nodes() []ResolvedNode {
	nodes := make([]ResolvedNode, 0, len(r.NodeMap))
	for _, node := range r.NodeMap {
		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsStruct != nodes[j].IsStruct {
			return !nodes[i].IsStruct
		}

		return strings.Compare(nodes[i].Key, nodes[j].Key) < 0
	})

	return nodes
}

type ResolvedNode struct {
	Spec
	IsStruct    bool
	ResolverKey string
	Type        reflect.Type
	Path        []string
	PathString  string
	StructKey   string
	Value       any
	ValueRaw    any
	ChildMap    map[string]ResolvedNode
}

func (r ResolvedNode) Children() []ResolvedNode {
	children := make([]ResolvedNode, 0, len(r.ChildMap))
	for _, child := range r.ChildMap {
		children = append(children, child)
	}

	sort.Slice(children, func(i, j int) bool {
		if children[i].IsStruct != children[j].IsStruct {
			return !children[i].IsStruct
		}

		return strings.Compare(children[i].Key, children[j].Key) < 0
	})

	return children
}

type StructResolver struct {
	resolvers []Resolver
	validate  *validator.Validate
}

func newStructResolver(resolvers []Resolver, validate *validator.Validate) *StructResolver {
	return &StructResolver{resolvers: resolvers, validate: validate}
}

func Resolve(resolvers []Resolver, validate *validator.Validate, specs []SpecEntry) (*ResolvedConfig, error) {
	resolved := &ResolvedConfig{
		NodeMap: make(map[string]ResolvedNode),
	}

	for _, spec := range specs {
		for _, opt := range spec.ValidatorOpts {
			opt(validate)
		}

		resolvedNode, resolveErr := resolveRootNode(
			sortResolvers(resolvers),
			validate,
			Spec{Key: spec.Key, DefaultValue: spec.DefaultValue},
		)
		if resolveErr != nil {
			return nil, resolveErr
		}

		resolved.NodeMap[spec.Key] = resolvedNode
	}

	return resolved, nil
}

func sortResolvers(resolvers []Resolver) []Resolver {
	sorted := make([]Resolver, len(resolvers))
	copy(sorted, resolvers)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority() == sorted[j].Priority() {
			return strings.Compare(sorted[i].Key(), sorted[j].Key()) < 0
		}

		return sorted[i].Priority() < sorted[j].Priority()
	})

	return sorted
}

func resolveRootNode(
	resolvers []Resolver,
	val *validator.Validate,
	spec Spec,
) (ResolvedNode, error) {
	path := strings.Split(spec.Key, ".")
	sr := newStructResolver(resolvers, val)

	return sr.resolveStruct(path[:len(path)-1], path[len(path)-1], "", reflect.ValueOf(spec.DefaultValue))
}

func (sr *StructResolver) resolveStruct(
	parentPath []string,
	key string,
	structKey string,
	value reflect.Value,
) (ResolvedNode, error) {
	if value.Type().Kind() != reflect.Struct {
		return ResolvedNode{}, fmt.Errorf("default value for key %q must be a struct, got %v", key, value.Type())
	}

	thisPath := copyAppend(parentPath, key)
	defaultValue := value.Interface()
	children := make(map[string]ResolvedNode)

	for i := range value.Type().NumField() {
		field := value.Type().Field(i)
		if !field.IsExported() {
			continue
		}

		fieldKey := strcase.ToSnake(field.Name)
		fieldValue := value.FieldByName(field.Name)

		resolved, err := sr.resolveField(thisPath, fieldKey, field.Name, field.Type, fieldValue)
		if err != nil {
			return ResolvedNode{}, fmt.Errorf("field %q: %w", fieldKey, err)
		}

		children[fieldKey] = resolved
	}

	resolvedValue, err := rebuildStruct(value.Type(), children)
	if err != nil {
		return ResolvedNode{}, fmt.Errorf("rebuild struct %q: %w", key, err)
	}

	if err := sr.validate.Struct(resolvedValue.Interface()); err != nil {
		return ResolvedNode{}, fmt.Errorf("validate %q: %w", strings.Join(thisPath, "."), err)
	}

	valueMap := make(map[string]any, len(children))
	for _, c := range children {
		valueMap[c.StructKey] = c.ValueRaw
	}

	return ResolvedNode{
		Spec: Spec{
			Key:          key,
			DefaultValue: defaultValue,
		},
		Path:       thisPath,
		PathString: strings.Join(thisPath, "."),
		StructKey:  structKey,
		IsStruct:   true,
		Type:       value.Type(),
		Value:      reflect.Indirect(resolvedValue).Interface(),
		ValueRaw:   valueMap,
		ChildMap:   children,
	}, nil
}

func (sr *StructResolver) resolveField(
	parentPath []string,
	fieldKey string,
	structKey string,
	fieldType reflect.Type,
	fieldValue reflect.Value,
) (ResolvedNode, error) {
	switch fieldType.Kind() {
	case reflect.Struct:
		return sr.resolveStruct(parentPath, fieldKey, structKey, fieldValue)

	case reflect.Slice:
		if fieldType.Elem().Kind() == reflect.Struct {
			return sr.resolveStructSlice(parentPath, fieldKey, structKey, fieldType, fieldValue)
		}

		fallthrough

	default:
		return sr.resolveScalarField(parentPath, fieldKey, structKey, fieldType, fieldValue)
	}
}

func (sr *StructResolver) resolveStructSlice(
	parentPath []string,
	fieldKey string,
	structKey string,
	fieldType reflect.Type,
	fieldValue reflect.Value,
) (ResolvedNode, error) {
	dv := fieldValue.Interface()

	for _, resolver := range sr.resolvers {
		if resolved, ok, err := resolver.Resolve(copyAppend(parentPath, fieldKey), fieldType); err != nil {
			return ResolvedNode{}, fmt.Errorf("resolve slice %q: %w", fieldKey, err)
		} else if ok {
			return ResolvedNode{
				Spec:        Spec{Key: fieldKey, DefaultValue: dv},
				ResolverKey: resolver.Key(),
				Type:        fieldType,
				Path:        parentPath,
				PathString:  strings.Join(copyAppend(parentPath, fieldKey), "."),
				StructKey:   structKey,
				Value:       resolved,
				ValueRaw:    resolved,
			}, nil
		}
	}

	length := fieldValue.Len()
	resolvedSlice := reflect.MakeSlice(fieldType, length, length)

	for i := range length {
		elemValue := fieldValue.Index(i)

		elemNode, err := sr.resolveStruct(parentPath, fieldKey, structKey, elemValue)
		if err != nil {
			return ResolvedNode{}, fmt.Errorf("slice element [%d]: %w", i, err)
		}

		resolvedSlice.Index(i).Set(reflect.ValueOf(elemNode.Value))
	}

	return ResolvedNode{
		Spec:       Spec{Key: fieldKey, DefaultValue: dv},
		Type:       fieldType,
		Path:       parentPath,
		PathString: strings.Join(copyAppend(parentPath, fieldKey), "."),
		StructKey:  structKey,
		Value:      resolvedSlice.Interface(),
		ValueRaw:   resolvedSlice.Interface(),
	}, nil
}

func (sr *StructResolver) resolveScalarField(
	parentPath []string,
	fieldKey string,
	structKey string,
	fieldType reflect.Type,
	fieldValue reflect.Value,
) (ResolvedNode, error) {
	dv := fieldValue.Interface()
	rv := dv

	var rk string

	for _, resolver := range sr.resolvers {
		if resolved, ok, err := resolver.Resolve(copyAppend(parentPath, fieldKey), fieldType); err != nil {
			return ResolvedNode{}, fmt.Errorf("resolve %q: %w", fieldKey, err)
		} else if ok {
			rv = resolved
			rk = resolver.Key()

			break
		}
	}

	return ResolvedNode{
		Spec:        Spec{Key: fieldKey, DefaultValue: dv},
		ResolverKey: rk,
		Type:        fieldType,
		Path:        parentPath,
		PathString:  strings.Join(copyAppend(parentPath, fieldKey), "."),
		StructKey:   structKey,
		Value:       rv,
		ValueRaw:    rv,
	}, nil
}

func copyAppend(base []string, extra string) []string {
	out := make([]string, len(base)+1)
	copy(out, base)
	out[len(base)] = extra

	return out
}

func rebuildStruct(structType reflect.Type, children map[string]ResolvedNode) (reflect.Value, error) {
	valueMap := make(map[string]any, len(children))
	for _, c := range children {
		valueMap[c.StructKey] = c.ValueRaw
	}

	resolvedValue := reflect.New(structType)

	decoder, decoderErr := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   resolvedValue.Interface(),
		MatchName: func(mapKey, fieldName string) bool {
			return mapKey == fieldName
		},
	})
	if decoderErr != nil {
		return reflect.Value{}, decoderErr
	}

	if decodeErr := decoder.Decode(valueMap); decodeErr != nil {
		return reflect.Value{}, decodeErr
	}

	return resolvedValue, nil
}
