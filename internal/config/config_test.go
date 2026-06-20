package config

import (
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortResolvers(t *testing.T) {
	t.Parallel()

	r1 := NewEnv(map[string]string{"A": "1"}, WithKey("z"), WithPriority(10))
	r2 := NewEnv(map[string]string{"B": "2"}, WithKey("a"), WithPriority(10))
	r3 := NewEnv(map[string]string{"C": "3"}, WithKey("m"), WithPriority(5))

	sorted := sortResolvers([]Resolver{r1, r2, r3})
	assert.Equal(t, 5, sorted[0].Priority())
	assert.Equal(t, 10, sorted[1].Priority())
	assert.Equal(t, 10, sorted[2].Priority())
	assert.Equal(t, "a", sorted[1].Key())
	assert.Equal(t, "z", sorted[2].Key())
}

func TestResolveStruct_ScalarFields(t *testing.T) {
	t.Parallel()

	type Simple struct {
		Name  string
		Count int
		Ratio float64
		Flag  bool
	}

	val := validator.New()
	resolvers := []Resolver{
		NewEnv(map[string]string{
			"SIMPLE_NAME":  "overridden",
			"SIMPLE_COUNT": "99",
		}, WithPriority(10)),
	}

	defaultCfg := Simple{Name: "default", Count: 1, Ratio: 0.5, Flag: true}

	node, err := resolveRootNode(resolvers, val, Spec{Key: "simple", DefaultValue: defaultCfg})
	require.NoError(t, err)

	resolved := node.Value.(Simple)
	assert.Equal(t, "overridden", resolved.Name)
	assert.Equal(t, 99, resolved.Count)
	assert.InEpsilon(t, 0.5, resolved.Ratio, 1e-9)
	assert.True(t, resolved.Flag)
}

func TestResolveStruct_ValidationFailure(t *testing.T) {
	t.Parallel()

	type Validated struct {
		Value string `validate:"uppercase"`
	}

	val := validator.New()
	resolvers := []Resolver{
		NewEnv(map[string]string{
			"VALIDATED_VALUE": "lowercase",
		}, WithPriority(10)),
	}

	_, err := resolveRootNode(resolvers, val, Spec{Key: "validated", DefaultValue: Validated{Value: "OK"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

func TestResolveStruct_UnknownResolverKey(t *testing.T) {
	t.Parallel()

	type Root struct {
		Field string
	}

	val := validator.New()
	node, err := resolveRootNode(nil, val, Spec{Key: "root", DefaultValue: Root{Field: "default"}})
	require.NoError(t, err)

	resolved := node.Value.(Root)
	assert.Equal(t, "default", resolved.Field)
}

func TestResolvedNode_ChildrenSorted(t *testing.T) {
	t.Parallel()

	val := validator.New()

	type Inner struct {
		ZField string
		AField int
	}

	type Outer struct {
		Inner Inner
	}

	node, err := resolveRootNode(nil, val, Spec{Key: "outer", DefaultValue: Outer{Inner: Inner{ZField: "z", AField: 1}}})
	require.NoError(t, err)

	children := node.Children()
	assert.Len(t, children, 1)

	innerNode := children[0]
	innerChildren := innerNode.Children()
	require.Len(t, innerChildren, 2)
	assert.Equal(t, "a_field", innerChildren[0].Key)
	assert.Equal(t, "z_field", innerChildren[1].Key)
}

func TestNodes_Sorted(t *testing.T) {
	t.Parallel()

	val := validator.New()

	type CfgA struct{ Val string }

	type CfgB struct{ Val int }

	cfg := ResolvedConfig{
		NodeMap: make(map[string]ResolvedNode),
	}

	nodeA, err := resolveRootNode(nil, val, Spec{Key: "zzz", DefaultValue: CfgA{Val: "a"}})
	require.NoError(t, err)

	cfg.NodeMap["zzz"] = nodeA

	nodeB, err := resolveRootNode(nil, val, Spec{Key: "aaa", DefaultValue: CfgB{Val: 1}})
	require.NoError(t, err)

	cfg.NodeMap["aaa"] = nodeB

	nodes := cfg.Nodes()
	assert.Len(t, nodes, 2)
	assert.Equal(t, "aaa", nodes[0].Key)
	assert.Equal(t, "zzz", nodes[1].Key)
}

func TestResolve(t *testing.T) {
	t.Parallel()

	type Cfg struct {
		Key   string
		Count int
	}

	val := validator.New()
	resolvers := []Resolver{
		NewEnv(map[string]string{
			"CFG_KEY": "from_env",
		}, WithPriority(10)),
	}

	resolved, err := Resolve(resolvers, val, []SpecEntry{
		{Key: "cfg", DefaultValue: Cfg{Key: "default", Count: 1}},
	})
	require.NoError(t, err)

	node, ok := resolved.NodeMap["cfg"]
	require.True(t, ok)

	cfg, ok := node.Value.(Cfg)
	require.True(t, ok)
	assert.Equal(t, "from_env", cfg.Key)
	assert.Equal(t, 1, cfg.Count)
}

func TestResolve_MultipleSpecs(t *testing.T) {
	t.Parallel()

	type CfgA struct {
		Val string
	}

	type CfgB struct {
		Num int
	}

	val := validator.New()
	resolved, err := Resolve(nil, val, []SpecEntry{
		{Key: "a", DefaultValue: CfgA{Val: "hello"}},
		{Key: "b", DefaultValue: CfgB{Num: 42}},
	})
	require.NoError(t, err)

	assert.Equal(t, "hello", resolved.NodeMap["a"].Value.(CfgA).Val)
	assert.Equal(t, 42, resolved.NodeMap["b"].Value.(CfgB).Num)
}

func TestResolve_ValidationTags(t *testing.T) {
	t.Parallel()

	type Cfg struct {
		Port int `validate:"gte=1,lte=65535"`
	}

	val := validator.New()
	_, err := Resolve(nil, val, []SpecEntry{
		{Key: "cfg", DefaultValue: Cfg{Port: 99999}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate")
}

func TestCopyAppend(t *testing.T) {
	t.Parallel()

	base := []string{"a", "b"}
	result := copyAppend(base, "c")

	assert.Equal(t, []string{"a", "b", "c"}, result)
	assert.Equal(t, []string{"a", "b"}, base)

	r2 := copyAppend(base, "d")

	assert.Equal(t, []string{"a", "b", "c"}, result)
	assert.Equal(t, []string{"a", "b", "d"}, r2)
}

func TestResolveStruct_SliceOfStructs(t *testing.T) {
	t.Parallel()

	type Item struct {
		Name  string
		Value int
	}

	type Container struct {
		Items []Item
	}

	val := validator.New()
	defaultCfg := Container{
		Items: []Item{
			{Name: "one", Value: 1},
			{Name: "two", Value: 2},
		},
	}

	node, err := resolveRootNode(nil, val, Spec{Key: "container", DefaultValue: defaultCfg})
	require.NoError(t, err)

	resolved := node.Value.(Container)
	require.Len(t, resolved.Items, 2)
	assert.Equal(t, "one", resolved.Items[0].Name)
	assert.Equal(t, 1, resolved.Items[0].Value)
	assert.Equal(t, "two", resolved.Items[1].Name)
	assert.Equal(t, 2, resolved.Items[1].Value)
}

func TestResolveStruct_TimeDuration(t *testing.T) {
	t.Parallel()

	type Config struct {
		Timeout time.Duration
	}

	val := validator.New()
	resolvers := []Resolver{
		NewEnv(map[string]string{
			"CONFIG_TIMEOUT": "30s",
		}, WithPriority(10)),
	}

	node, err := resolveRootNode(resolvers, val, Spec{Key: "config", DefaultValue: Config{Timeout: time.Second}})
	require.NoError(t, err)

	resolved := node.Value.(Config)
	assert.Equal(t, 30*time.Second, resolved.Timeout)
}

func TestResolveScalarField_NoResolver(t *testing.T) {
	t.Parallel()

	type Config struct {
		Name string
		Val  int
	}

	val := validator.New()
	node, err := resolveRootNode(nil, val, Spec{Key: "cfg", DefaultValue: Config{Name: "test", Val: 42}})
	require.NoError(t, err)

	resolved := node.Value.(Config)
	assert.Equal(t, "test", resolved.Name)
	assert.Equal(t, 42, resolved.Val)
}
