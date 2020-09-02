package aorm

var DefaultM2MNamer = func(field *StructField) func(singular bool) string {
	return func(singular bool) string {
		if tag := field.TagSettings.GetTags("M2M"); tag != nil {
			if singular {
				return tag["S"]
			} else {
				return tag["P"]
			}
		}
		return field.TagSettings["M2M"]
	}
}
