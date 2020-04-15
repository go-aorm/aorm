package aorm

func AbbrToTextType(abbr string) string {
	switch abbr {
	case "tiny", "small":
		return "VARCHAR(127)"
	case "medium":
		return "VARCHAR(512)"
	case "large":
		return "VARCHAR(1024)"
	default:
		return ""
	}
}
