package layout

import "testing"

func TestTruncateKpiScope(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		max   int
		want  string
	}{
		{name: "short", scope: "dag_a", max: 20, want: "dag_a"},
		{name: "long", scope: "very_long_dag_identifier", max: 12, want: "very_long..."},
		{name: "tiny", scope: "abcdef", max: 2, want: "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateKpiScope(tt.scope, tt.max)
			if got != tt.want {
				t.Fatalf("truncateKpiScope(%q, %d)=%q want %q", tt.scope, tt.max, got, tt.want)
			}
		})
	}
}
