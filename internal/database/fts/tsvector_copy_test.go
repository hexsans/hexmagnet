package fts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTsvectorCopy_Empty(t *testing.T) {
	t.Parallel()

	original := Tsvector{}
	copied := original.Copy()
	assert.Equal(t, Tsvector{}, copied)
	copied["new"] = map[int]TsvectorWeight{1: 'A'}
	_, exists := original["new"]
	assert.False(t, exists, "modifying copy must not affect original")
}

func TestTsvectorCopy_ShallowCopyPrevention(t *testing.T) {
	t.Parallel()

	original := Tsvector{
		"foo": {1: 'A', 2: 'B'},
		"bar": {3: 'C'},
	}
	copied := original.Copy()
	assert.Equal(t, original, copied)

	original["foo"][1] = 'D'
	assert.NotEqual(t, original, copied)

	copied["bar"][3] = 'D'
	assert.NotEqual(t, original, copied)

	original["new"] = map[int]TsvectorWeight{4: 'A'}
	_, exists := copied["new"]
	assert.False(t, exists)
}

func TestTsvectorCopy_SingleEntry(t *testing.T) {
	t.Parallel()

	original := Tsvector{
		"test": {1: 'A'},
	}
	copied := original.Copy()
	assert.Equal(t, original, copied)
	copied["test"][1] = 'B'
	assert.NotEqual(t, original, copied)
}

func TestTsvectorCopy_EmptyInnerMap(t *testing.T) {
	t.Parallel()

	original := Tsvector{
		"empty": {},
	}
	copied := original.Copy()
	assert.Equal(t, original, copied)
	original["empty"][1] = 'A'
	assert.NotEqual(t, original, copied)
}
