package testutil

import "github.com/hexsans/hexmagnet/internal/protocol"

func MustParseID(s string) protocol.ID {
	id, err := protocol.ParseID(s)
	if err != nil {
		panic(err)
	}

	return id
}
