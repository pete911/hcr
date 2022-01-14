package utils

func SecretValue(in string) string {
	if len(in) == 0 {
		return "<empty>"
	}
	return "*****"
}
