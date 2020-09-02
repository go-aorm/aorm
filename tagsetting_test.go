package aorm_test

import (
	"reflect"
	"testing"

	"github.com/moisespsena-go/aorm"
)

func TestTagSetting_ParseString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want aorm.TagSetting
	}{
		{"simple", `fkc;a:b;c:{d;e:f;g:{g;h:i}};j:{k};l:{m}`, aorm.TagSetting{"FKC": "FKC", "A": "b", "C": "{d;e:f;g:{g;h:i}}", "J": "{k}", "L": "{m}"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tag aorm.TagSetting
			tag.ParseString(tt.s)
			if !reflect.DeepEqual(tag, tt.want) {
				t.Errorf("expected %v; but got = %v", tt.want, tag)
			}
		})
	}
}
