package protocgenghe

func EscapeTypescriptString(s string) string {
	b := []rune(s)
	result := make([]rune, 0, len(b))
	for _, ch := range b {
		switch ch {
		case '\'':
			result = append(result, '\\', '\'')
		case '"':
			result = append(result, '\\', '"')
		case '\\':
			result = append(result, '\\', '\\')
		case '\n':
			result = append(result, '\\', 'n')
		case '\t':
			result = append(result, '\\', 't')
		case '`':
			result = append(result, '\\', 'x', '6', '0')
		case '$':
			result = append(result, '\\', 'x', '2', '4')
		default:
			result = append(result, ch)
		}
	}
	return string(result)
}
