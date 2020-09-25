package aorm

import (
	"strings"

	tag_scanner "github.com/unapu-go/tag-scanner"
)

type TagSetting map[string]string

func (this TagSetting) Clone() (clone TagSetting) {
	if this == nil {
		return
	}
	clone = make(TagSetting)
	for key, value := range this {
		clone[key] = value
	}
	return
}

func (this TagSetting) Get(key string) (v string) {
	if this != nil {
		v = this[key]
	}
	return
}

func (this TagSetting) GetOk(key string) (v string, ok bool) {
	if this == nil {
		return
	}
	v, ok = this[key]
	return
}

func (this TagSetting) GetString(key string) (v string) {
	if this == nil {
		return
	}
	if v, ok := this[key]; ok {
		return tag_scanner.Default.String(v)
	}
	return
}

func (this TagSetting) GetStringAlias(key string, alias ...string) (v string) {
	if this == nil {
		return
	}
	if v, ok := this[key]; ok {
		return tag_scanner.Default.String(v)
	}
	for _, key := range alias {
		if v, ok := this[key]; ok {
			return tag_scanner.Default.String(v)
		}
	}
	return
}

func (this TagSetting) Empty() bool {
	return this == nil || len(this) == 0
}

func (this TagSetting) Enable(key string) {
	this[key] = key
}

func (this *TagSetting) Set(key, value string) {
	if *this == nil {
		*this = make(TagSetting)
	}
	(*this)[key] = value
}

func (this TagSetting) Flag(name string) bool {
	if this == nil {
		return false
	}
	return this[name] == name
}

func (this TagSetting) String() string {
	var pairs []string
	for k, v := range this {
		if k == v {
			pairs = append(pairs, k)
		} else {
			pairs = append(pairs, k+":"+v)
		}
	}
	return strings.Join(pairs, "; ")
}

func (this TagSetting) Update(setting ...map[string]string) {
	for _, setting := range setting {
		for k, v := range setting {
			this[k] = v
		}
	}
}

func (this TagSetting) GetTags(name string) (tags TagSetting) {
	if s := this[name]; s != "" && this.Scanner().IsTags(s) {
		tags = make(TagSetting)
		tags.ParseString(s)
	}
	return
}

func (this TagSetting) TagsOf(value string) (tags TagSetting) {
	if value != "" && this.Scanner().IsTags(value) {
		tags = make(TagSetting)
		tags.ParseString(value)
	}
	return
}

func (this *TagSetting) Parse(tags StructTag, key string, keyAlias ...string) (ok bool) {
	return this.ParseCallback(tags, append([]string{key}, keyAlias...))
}

func (this *TagSetting) ParseCallback(tags StructTag, keys []string, cb ...func(dest map[string]string, n tag_scanner.Node)) (ok bool) {
	if *this == nil {
		*this = make(TagSetting)
	}
	var tags_ = make([]string, len(keys))
	for i, key := range keys {
		tags_[i] = tags.Get(key)
	}
	for _, str := range tags_ {
		(*this).ParseString(str)
	}
	return len(*this) > 0
}

func (this *TagSetting) Scanner() tag_scanner.Scanner {
	return tag_scanner.Default
}

func (this *TagSetting) ParseString(s string, cb ...func(dest map[string]string, n tag_scanner.Node)) {
	if *this == nil {
		*this = map[string]string{}
	}
	scanner := this.Scanner()
	scanner.ScanAll(s, func(node tag_scanner.Node) {
		for _, cb := range cb {
			cb(*this, node)
		}
		switch node.Type() {
		case tag_scanner.Tags:
			// pass
		case tag_scanner.KeyValue:
			kv := node.(tag_scanner.NodeKeyValue)
			(*this)[strings.ToUpper(kv.Key)] = kv.Value
		case tag_scanner.Flag:
			name := strings.ToUpper(node.String())
			(*this)[name] = name
		}
	})
}
