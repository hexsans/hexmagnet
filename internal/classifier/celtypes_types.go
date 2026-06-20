package classifier

func nullableString(s string, valid bool) any {
	if !valid {
		return nil
	}

	return s
}

func nullableUint(v uint, valid bool) any {
	if !valid {
		return nil
	}

	return int64(v)
}
