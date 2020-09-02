package aorm

// NamifyString Humanize separates string based on capitalizd letters
// e.g. "order_item-data" -> "OrderItemData"
func NamifyString(s string) string {
	var human []rune
	var toUpper bool
	s = "_" + s
	for _, c := range s {
		if c == '_' || c == '-' {
			toUpper = true
			continue
		} else if c == '!' {
			toUpper = true
		} else if toUpper {
			toUpper = false
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}
		}
		human = append(human, c)
	}
	return string(human)
}
