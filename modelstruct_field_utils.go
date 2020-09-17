package aorm

var DefaultM2MNamer = func(field *StructField) JoinTableHandlerNamer {
	return func(singular bool, _, _ *JoinTableSource) string {
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
