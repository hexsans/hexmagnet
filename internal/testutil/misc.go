package testutil

func StrPtr(v string) *string { return &v }

func ValidHash() string {
	return "0102030405060708090a0b0c0d0e0f1011121314"
}
