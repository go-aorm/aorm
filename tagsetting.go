package aorm

type TagSetting map[string]string

func (this TagSetting) Enable(key string) {
	this[key] = key
}

func (this TagSetting) Set(key, value string) {
	this[key] = value
}

func (this TagSetting) Flag(name string) bool {
	if this == nil {
		return false
	}
	return this[name] != ""
}

func (this TagSetting) Update(setting ...map[string]string) {
	for _, setting := range setting {
		for k, v := range setting {
			this[k] = v
		}
	}
}
