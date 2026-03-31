package template

import (
	"testing"
	"time"
)

func TestEngine_Execute(t *testing.T) {
	e := New()

	tests := []struct {
		name    string
		tmpl    string
		data    interface{}
		want    string
		wantErr bool
	}{
		{
			name: "plain text",
			tmpl: "hello world",
			data: nil,
			want: "hello world",
		},
		{
			name: "data field",
			tmpl: "hello {{.Name}}",
			data: struct{ Name string }{"world"},
			want: "hello world",
		},
		{
			name: "now function",
			tmpl: "{{now.Format \"2006-01-02\"}}",
			data: nil,
			want: time.Now().Format("2006-01-02"),
		},
		{
			name:    "invalid template",
			tmpl:    "hello {{.NoExist}",
			data:    nil,
			wantErr: true,
		},
		{
			name: "upper",
			tmpl: "{{upper .}}",
			data: "hello",
			want: "HELLO",
		},
		{
			name: "lower",
			tmpl: "{{lower .}}",
			data: "HELLO",
			want: "hello",
		},
		{
			name: "trim",
			tmpl: "{{trim .}}",
			data: "  hello  ",
			want: "hello",
		},
		{
			name: "replace",
			tmpl: "{{replace \"l\" \"w\" .}}",
			data: "hello",
			want: "hewwo",
		},
		{
			name: "split and join",
			tmpl: "{{join \",\" (split \" \" .)}}",
			data: "a b c",
			want: "a,b,c",
		},
		{
			name: "contains true",
			tmpl: "{{contains \"world\" .}}",
			data: "hello world",
			want: "true",
		},
		{
			name: "contains false",
			tmpl: "{{contains \"foo\" .}}",
			data: "hello world",
			want: "false",
		},
		{
			name: "regexMatch true",
			tmpl: "{{regexMatch \"^h.*o$\" .}}",
			data: "hello",
			want: "true",
		},
		{
			name: "regexFind",
			tmpl: "{{regexFind \"[a-z]+\" .}}",
			data: "123 abc 456",
			want: "abc",
		},
		{
			name: "regexReplace",
			tmpl: "{{regexReplace \"[0-9]+\" \"#\" .}}",
			data: "123 abc 456",
			want: "# abc #",
		},
		{
			name: "shellEscape",
			tmpl: "{{shellEscape .}}",
			data: "hello world's",
			want: "'hello world'\\''s'",
		},
		{
			name: "urlEncode",
			tmpl: "{{urlEncode .}}",
			data: "hello world",
			want: "hello+world",
		},
		{
			name: "quote",
			tmpl: "{{quote .}}",
			data: "hello",
			want: `"hello"`,
		},
		{
			name: "truncate",
			tmpl: "{{truncate 5 .}}",
			data: "hello world",
			want: "hello...",
		},
		{
			name: "padLeft",
			tmpl: "{{padLeft 10 .}}",
			data: "hello",
			want: "     hello",
		},
		{
			name: "padRight",
			tmpl: "{{padRight 10 .}}",
			data: "hello",
			want: "hello     ",
		},
		{
			name: "indent",
			tmpl: "{{indent 2 .}}",
			data: "line1\nline2",
			want: "  line1\n  line2",
		},
		{
			name: "chaining",
			tmpl: "{{ . | trim | upper | truncate 3 }}",
			data: "   hello world   ",
			want: "HEL...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.Execute(tt.name, tt.tmpl, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Engine.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
