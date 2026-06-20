package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsertMap_New_Empty(t *testing.T) {
	t.Parallel()

	m := NewInsertMap[string, int]()
	assert.Empty(t, m.Entries())
}

func TestInsertMap_New_WithEntries(t *testing.T) {
	t.Parallel()

	m := NewInsertMap(MapEntry[string, int]{Key: "a", Value: 1}, MapEntry[string, int]{Key: "b", Value: 2})
	assert.Len(t, m.Entries(), 2)
}

func TestInsertMap_Set_InsertionOrder(t *testing.T) {
	t.Parallel()

	m := NewInsertMap[string, int]()
	m.Set("z", 100)
	m.Set("a", 1)
	m.Set("m", 50)

	entries := m.Entries()
	assert.Equal(t, []string{"z", "a", "m"}, []string{entries[0].Key, entries[1].Key, entries[2].Key})
	assert.Equal(t, []int{100, 1, 50}, []int{entries[0].Value, entries[1].Value, entries[2].Value})
}

func TestInsertMap_Set_OverwriteExisting(t *testing.T) {
	t.Parallel()

	m := NewInsertMap[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("a", 999)

	entries := m.Entries()
	assert.Len(t, entries, 2, "key position should be preserved")
	assert.Equal(t, []string{"a", "b"}, []string{entries[0].Key, entries[1].Key})
	assert.Equal(t, []int{999, 2}, []int{entries[0].Value, entries[1].Value})
}

func TestInsertMap_SetEntries(t *testing.T) {
	t.Parallel()

	m := NewInsertMap[string, int]()
	m.SetEntries(
		MapEntry[string, int]{Key: "x", Value: 10},
		MapEntry[string, int]{Key: "y", Value: 20},
	)

	entries := m.Entries()
	assert.Len(t, entries, 2)
	assert.Equal(t, "x", entries[0].Key)
	assert.Equal(t, "y", entries[1].Key)
}

func TestInsertMap_Entries_CorrectOrder(t *testing.T) {
	t.Parallel()

	m := NewInsertMap[string, string]()
	m.Set("k1", "v1")
	m.Set("k2", "v2")

	entries := m.Entries()
	assert.Len(t, entries, 2)
	assert.Equal(t, "k1", entries[0].Key)
	assert.Equal(t, "v1", entries[0].Value)
	assert.Equal(t, "k2", entries[1].Key)
	assert.Equal(t, "v2", entries[1].Value)
}
