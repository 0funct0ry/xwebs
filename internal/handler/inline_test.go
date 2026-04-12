package handler

import (
	"reflect"
	"testing"
)

func TestParseInlineHandler(t *testing.T) {
	tests := []struct {
		name           string
		hStr           string
		defaultRespond string
		index          int
		want           Handler
		wantErr        bool
	}{
		{
			name:  "simple glob match and respond",
			hStr:  "ping :: respond:pong",
			index: 1,
			want: Handler{
				Name: "inline-1",
				Match: Matcher{
					Type:    "glob",
					Pattern: "ping",
				},
				Respond: "pong",
			},
		},
		{
			name:  "jq match and run",
			hStr:  ".type == \"ping\" :: run:echo pong",
			index: 2,
			want: Handler{
				Name: "inline-2",
				Match: Matcher{
					Type:    "jq",
					Pattern: ".type == \"ping\"",
				},
				Run: "echo pong",
			},
		},
		{
			name:  "regex match with actions and options",
			hStr:  "^CMD: :: run:dispatch.sh :: timeout:5s :: exclusive",
			index: 3,
			want: Handler{
				Name: "inline-3",
				Match: Matcher{
					Type:    "regex",
					Pattern: "^CMD:",
				},
				Run:       "dispatch.sh",
				Timeout:   "5s",
				Exclusive: true,
			},
		},
		{
			name:           "run with default respond",
			hStr:           "test :: run:cmd.sh",
			defaultRespond: "done",
			index:          4,
			want: Handler{
				Name: "inline-4",
				Match: Matcher{
					Type:    "glob",
					Pattern: "test",
				},
				Run:     "cmd.sh",
				Respond: "done",
			},
		},
		{
			name:           "run with explicit respond overrides default",
			hStr:           "test :: run:cmd.sh :: respond:yay",
			defaultRespond: "done",
			index:          5,
			want: Handler{
				Name: "inline-5",
				Match: Matcher{
					Type:    "glob",
					Pattern: "test",
				},
				Run:     "cmd.sh",
				Respond: "yay",
			},
		},
		{
			name:  "explicit prefix glob",
			hStr:  "glob:^start* :: run:handle.sh",
			index: 6,
			want: Handler{
				Name: "inline-6",
				Match: Matcher{
					Type:    "glob",
					Pattern: "^start*",
				},
				Run: "handle.sh",
			},
		},
		{
			name:  "match shorthand *",
			hStr:  "* :: respond:echo",
			index: 7,
			want: Handler{
				Name: "inline-7",
				Match: Matcher{
					Type:    "glob",
					Pattern: "*",
				},
				Respond: "echo",
			},
		},
		{
			name:    "invalid segment",
			hStr:    "test :: unknown:cmd",
			index:   8,
			wantErr: true,
		},
		{
			name:    "missing action",
			hStr:    "test :: exclusive",
			index:   9,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInlineHandler(tt.hStr, tt.defaultRespond, tt.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInlineHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseInlineHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAutoDetectMatcher(t *testing.T) {
	tests := []struct {
		expr string
		want Matcher
	}{
		{".type", Matcher{Type: "jq", Pattern: ".type"}},
		{"*error*", Matcher{Type: "glob", Pattern: "*error*"}},
		{"^CMD", Matcher{Type: "regex", Pattern: "^CMD"}},
		{"(ping|pong)", Matcher{Type: "regex", Pattern: "(ping|pong)"}},
		{"literal", Matcher{Type: "glob", Pattern: "literal"}},
		{"jq:true", Matcher{Type: "jq", Pattern: "true"}},
		{"glob:^start*", Matcher{Type: "glob", Pattern: "^start*"}},
		{"*", Matcher{Type: "glob", Pattern: "*"}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got := AutoDetectMatcher(tt.expr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AutoDetectMatcher(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}
