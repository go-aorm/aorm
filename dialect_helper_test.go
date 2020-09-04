package aorm

import "testing"

func TestTextSize(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want uint16
	}{
		{"a", "varchar", 0},
		{"a", "varchar(2)", 2},
		{"a", "varchar(289)", 289},
		{"a", "varchar  (289)", 289},
		{"a", "char  (289)", 289},
		{"a", "tiny", 127},
		{"a", "small", 127},
		{"a", "medium", 512},
		{"a", "large", 1024},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TextSize(tt.typ); got != tt.want {
				t.Errorf("TextSize() = %v, want %v", got, tt.want)
			}
		})
	}
}