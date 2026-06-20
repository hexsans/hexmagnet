package resolvers

func ptr(s string) *string { return &s }

func nilStr(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
