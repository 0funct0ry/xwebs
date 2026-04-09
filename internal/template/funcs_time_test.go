package template

import (
	"regexp"
	"testing"
	"time"
)

func TestTimeFuncs(t *testing.T) {
	e := New(false)

	// We use a margin for time comparison to avoid flakes
	now := time.Now()

	tests := []struct {
		name string
		tmpl string
		want string // if non-empty, exact match
		isRE bool   // if true, treat want as regex
	}{
		{
			name: "time",
			tmpl: "{{time}}",
			want: `^\d{2}:\d{2}:\d{2}$`,
			isRE: true,
		},
		{
			name: "shortTime",
			tmpl: "{{shortTime}}",
			want: `^\d{2}:\d{2}$`,
			isRE: true,
		},
		{
			name: "date",
			tmpl: "{{date}}",
			want: `^\d{4}-\d{2}-\d{2}$`,
			isRE: true,
		},
		{
			name: "weekday",
			tmpl: "{{weekday}}",
			want: now.Weekday().String(),
		},
		{
			name: "isoTime",
			tmpl: "{{isoTime}}",
			want: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})$`,
			isRE: true,
		},
		{
			name: "unix",
			tmpl: "{{unix}}",
			want: `^\d+$`,
			isRE: true,
		},
		{
			name: "unixMilli",
			tmpl: "{{unixMilli}}",
			want: `^\d+$`,
			isRE: true,
		},
		{
			name: "hour",
			tmpl: "{{hour}}",
			want: `^\d{2}$`,
			isRE: true,
		},
		{
			name: "minute",
			tmpl: "{{minute}}",
			want: `^\d{2}$`,
			isRE: true,
		},
		{
			name: "elapsed",
			tmpl: "{{elapsed}}",
			want: `^\d+s$|^\d+m\d+s$|^\d+h\d+m\d+s$`,
			isRE: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.Execute(tt.name, tt.tmpl, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.isRE {
				match := matchRegex(tt.want, got)
				if !match {
					t.Errorf("got %q, want regex %q", got, tt.want)
				}
			} else if tt.want != "" {
				if got != tt.want {
					// Weekday might change if test runs across midnight, but unlikely
					t.Errorf("got %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func matchRegex(pattern, s string) bool {
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}
