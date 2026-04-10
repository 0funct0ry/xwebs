package template

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestEngine_Execute(t *testing.T) {
	e := New(false)

	tests := []struct {
		name    string
		tmpl    string
		data    interface{}
		want    string
		wantErr bool
	}{
		// ... existing tests ...
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
		{
			name: "toJSON",
			tmpl: "{{toJSON .}}",
			data: map[string]string{"foo": "bar"},
			want: `{"foo":"bar"}`,
		},
		{
			name: "fromJSON",
			tmpl: "{{(fromJSON .).foo}}",
			data: `{"foo":"bar"}`,
			want: "bar",
		},
		{
			name: "jq",
			tmpl: "{{jq \".foo\" .}}",
			data: `{"foo":"bar"}`,
			want: "bar",
		},
		{
			name: "base64",
			tmpl: "{{base64Encode . | base64Decode}}",
			data: "hello world",
			want: "hello world",
		},
		{
			name: "hex",
			tmpl: "{{hexEncode . | hexDecode}}",
			data: "hello world",
			want: "hello world",
		},
		{
			name: "md5",
			tmpl: "{{md5 .}}",
			data: "hello",
			want: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name: "sha256",
			tmpl: "{{sha256 .}}",
			data: "hello",
			want: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name: "math add",
			tmpl: "{{add 1 2}}",
			data: nil,
			want: "3",
		},
		{
			name: "math seq",
			tmpl: "{{range seq 1 3}}{{.}}{{end}}",
			data: nil,
			want: "123",
		},
		{
			name: "collection dict",
			tmpl: "{{$d := dict \"a\" 1 \"b\" 2}}{{$d.a}}{{$d.b}}",
			data: nil,
			want: "12",
		},
		{
			name: "collection default",
			tmpl: "{{default \"foo\" .}}",
			data: "",
			want: "foo",
		},
		{
			name: "collection ternary",
			tmpl: "{{ternary true \"yes\" \"no\"}}",
			data: nil,
			want: "yes",
		},
		{
			name: "system env",
			tmpl: "{{env \"USER\"}}",
			data: nil,
			want: os.Getenv("USER"),
		},
		{
			name: "id uuid",
			tmpl: "{{uuid | len}}",
			data: nil,
			want: "36",
		},
		{
			name: "id counter",
			tmpl: "{{counter \"a\"}}{{counter \"a\"}}",
			data: nil,
			want: "12",
		},
		{
			name: "system user",
			tmpl: "{{user}}",
			data: nil,
			// want: verified via regex or existence
		},
		{
			name: "system home",
			tmpl: "{{home}}",
			data: nil,
		},
		{
			name: "system xwebsVersion",
			tmpl: "{{xwebsVersion}}",
			data: nil,
			want: "dev", // Default when not set by ldflags
		},
		{
			name: "system cpuUsage",
			tmpl: "{{cpuUsage}}",
			data: nil,
		},
		{
			name: "system memUsage",
			tmpl: "{{memUsage}}",
			data: nil,
		},
		{
			name: "system diskUsage",
			tmpl: "{{diskUsage}}",
			data: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.Execute(tt.name, tt.tmpl, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.name == "system user" || tt.name == "system home" || 
			   tt.name == "system cpuUsage" || tt.name == "system memUsage" || 
			   tt.name == "system diskUsage" {
				if got == "" {
					t.Errorf("%s returned empty string", tt.name)
				}
				return
			}
			if got != tt.want {
				t.Errorf("Engine.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_Sandbox(t *testing.T) {
	e := New(true)

	tests := []struct {
		name    string
		tmpl    string
		data    interface{}
		wantErr bool
	}{
		{
			name:    "env disabled",
			tmpl:    "{{env \"USER\"}}",
			wantErr: true,
		},
		{
			name:    "shell disabled",
			tmpl:    "{{shell \"whoami\"}}",
			wantErr: true,
		},
		{
			name:    "fileRead disabled",
			tmpl:    "{{fileRead \"/etc/passwd\"}}",
			wantErr: true,
		},
		{
			name:    "hostname disabled",
			tmpl:    "{{hostname}}",
			wantErr: true,
		},
		{
			name:    "user disabled",
			tmpl:    "{{user}}",
			wantErr: true,
		},
		{
			name:    "home disabled",
			tmpl:    "{{home}}",
			wantErr: true,
		},
		{
			name:    "cpuUsage disabled",
			tmpl:    "{{cpuUsage}}",
			wantErr: true,
		},
		{
			name:    "memUsage disabled",
			tmpl:    "{{memUsage}}",
			wantErr: true,
		},
		{
			name:    "diskUsage disabled",
			tmpl:    "{{diskUsage}}",
			wantErr: true,
		},
		{
			name:    "math works in sandbox",
			tmpl:    "{{add 1 2}}",
			wantErr: false,
		},
		{
			name:    "string works in sandbox",
			tmpl:    "{{upper \"hello\"}}",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.Execute(tt.name, tt.tmpl, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "is disabled in sandbox mode") {
					t.Errorf("Engine.Execute() error message = %v, want it to contain 'is disabled in sandbox mode'", err)
				}
				// Verify simplification - no boilerplate from text/template
				if strings.Contains(err.Error(), "template:") || strings.Contains(err.Error(), "executing") {
					t.Errorf("Engine.Execute() error message still contains boilerplate: %v", err)
				}
			}
		})
	}
}

func TestEngine_Context(t *testing.T) {
	e := New(false)

	ctx := NewContext()
	ctx.Conn = &ConnectionContext{
		URL:     "ws://example.com",
		Headers: map[string]string{"X-Test": "Value"},
	}
	ctx.Msg = &MessageContext{
		Type:   "text",
		Data:   "hello",
		Length: 5,
	}

	tests := []struct {
		name string
		tmpl string
		want string
	}{
		{
			name: "conn url",
			tmpl: "{{.Conn.URL}}",
			want: "ws://example.com",
		},
		{
			name: "conn header",
			tmpl: "{{index .Conn.Headers \"X-Test\"}}",
			want: "Value",
		},
		{
			name: "msg data",
			tmpl: "{{.Msg.Data}}",
			want: "hello",
		},
		{
			name: "session set and get",
			tmpl: "{{sessionSet \"foo\" \"bar\"}}{{sessionGet \"foo\"}}",
			want: "bar",
		},
		{
			name: "env access",
			tmpl: "{{index .Env \"TEST_VAR\"}}",
			want: "test_value",
		},
		{
			name: "mode and state indicators",
			tmpl: "{{mode}} | {{status}} | {{reconnectCount}} | {{port}} | {{tls}} | {{secure}}",
			want: "client | reconnecting | 5 | 8080 | 🔒 | true",
		},
		{
			name: "mode and state indicators default",
			tmpl: "{{mode}} | {{status}} | {{tls}} | {{secure}}",
			want: "server | closed |  | false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data interface{} = ctx
			if tt.name == "env access" {
				ctx.Env = map[string]string{"TEST_VAR": "test_value"}
			} else if tt.name == "mode and state indicators" {
				data = &TemplateContext{
					Mode:           "client",
					Status:         "reconnecting",
					ReconnectCount: 5,
					Port:           8080,
					IsSecure:       true,
				}
			} else if tt.name == "mode and state indicators default" {
				data = &TemplateContext{
					Mode: "server",
				}
			}

			got, err := e.Execute(tt.name, tt.tmpl, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
