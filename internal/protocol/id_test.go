package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantStr string
	}{
		{"valid hex", "0102030405060708090a0b0c0d0e0f1011121314", false, "0102030405060708090a0b0c0d0e0f1011121314"},
		{"with 0x prefix", "0x0102030405060708090a0b0c0d0e0f1011121314", false, "0102030405060708090a0b0c0d0e0f1011121314"},
		{"all zeros", "0000000000000000000000000000000000000000", false, "0000000000000000000000000000000000000000"},
		{"all max", "ffffffffffffffffffffffffffffffffffffffff", false, "ffffffffffffffffffffffffffffffffffffffff"},
		{"too short", "010203", true, ""},
		{"empty string", "", true, ""},
		{"invalid hex chars", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", true, ""},
		{"odd length", "0102030405060708090a0b0c0d0e0f101112131", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := ParseID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, id, 20)
			assert.Equal(t, tt.wantStr, id.String())
		})
	}
}

func TestRandomNodeID(t *testing.T) {
	t.Parallel()

	id := RandomNodeID()
	assert.Len(t, id, 20)
	assert.NotEqual(t, ID{}, id)

	// Verify different calls produce different IDs
	id2 := RandomNodeID()
	assert.NotEqual(t, id, id2)
}

func TestRandomNodeIDWithClientSuffix(t *testing.T) {
	t.Parallel()

	id := RandomNodeIDWithClientSuffix()
	assert.Len(t, id, 20)
	assert.NotEqual(t, ID{}, id)

	// Verify client suffix is at the end
	suffix := string(id[20-len(idClientPart):])
	assert.Equal(t, idClientPart, suffix)

	// Verify prefix is random (not overwritten by suffix)
	prefix := string(id[:20-len(idClientPart)])
	assert.NotEqual(t, idClientPart, prefix)

	// Verify different calls produce different IDs
	id2 := RandomNodeIDWithClientSuffix()
	assert.NotEqual(t, id, id2)
}

func TestRandomPeerID(t *testing.T) {
	t.Parallel()

	id := RandomPeerID()
	assert.Len(t, id, 20)
	assert.NotEqual(t, ID{}, id)

	// Verify client prefix is at the beginning
	prefix := string(id[:len(idClientPart)])
	assert.Equal(t, idClientPart, prefix)
}

func TestIDInt160(t *testing.T) {
	t.Parallel()

	id := mustParseID("0102030405060708090a0b0c0d0e0f1011121314")
	i160 := id.Int160()
	assert.Equal(t, id[:], i160.Bytes())
}

func TestIDIsZero(t *testing.T) {
	t.Parallel()
	assert.True(t, (ID{}).IsZero())
	assert.False(t, (mustParseID("0102030405060708090a0b0c0d0e0f1011121314")).IsZero())
}

func TestIDGetBit(t *testing.T) {
	t.Parallel()

	id := ID{0: 0b10000000}
	assert.True(t, id.GetBit(0))
	assert.False(t, id.GetBit(1))
}

func TestIDBytes(t *testing.T) {
	t.Parallel()

	id := mustParseID("0102030405060708090a0b0c0d0e0f1011121314")
	assert.Equal(t, id[:], id.Bytes())
	assert.Len(t, id.Bytes(), 20)
}

func TestIDString(t *testing.T) {
	t.Parallel()

	id := mustParseID("0102030405060708090a0b0c0d0e0f1011121314")
	assert.Equal(t, "0102030405060708090a0b0c0d0e0f1011121314", id.String())
}

func TestNewIDFromByteSlice(t *testing.T) {
	t.Parallel()

	valid := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	id, err := NewIDFromByteSlice(valid)
	require.NoError(t, err)
	assert.Equal(t, valid, id[:])

	_, err = NewIDFromByteSlice([]byte{1, 2, 3})
	assert.Error(t, err)
}

func TestIDMarshalUnmarshalBinary(t *testing.T) {
	t.Parallel()

	id := mustParseID("0102030405060708090a0b0c0d0e0f1011121314")
	b, err := id.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, id[:], b)

	var id2 ID

	err = id2.UnmarshalBinary(b)
	require.NoError(t, err)
	assert.Equal(t, id, id2)

	err = id2.UnmarshalBinary([]byte{1, 2, 3})
	assert.Error(t, err)
}

func TestIDBencodeRoundTrip(t *testing.T) {
	t.Parallel()

	id := mustParseID("0102030405060708090a0b0c0d0e0f1011121314")
	b, err := id.MarshalBencode()
	require.NoError(t, err)

	var id2 ID

	err = id2.UnmarshalBencode(b)
	require.NoError(t, err)
	assert.Equal(t, id, id2)
}

func mustParseID(s string) ID {
	id, err := ParseID(s)
	if err != nil {
		panic(err)
	}

	return id
}
