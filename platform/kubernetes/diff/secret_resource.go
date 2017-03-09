package diff

type Secret struct {
	baseObject
	Data map[string]SecretData
	Type string
}

type SecretData string

func (s SecretData) Diff(d Differ, path string) ([]Difference, error) {
	if s1, ok := d.(SecretData); ok {
		if s1 == s {
			return nil, nil
		}
		return []Difference{opaqueChanged{path}}, nil
	}
	return nil, ErrNotDiffable
}
