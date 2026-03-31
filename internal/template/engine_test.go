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
