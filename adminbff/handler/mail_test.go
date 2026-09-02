package handler

import "testing"

func TestNormalizeOptionalParameterSchema(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty", input: "", want: ""},
		{name: "whitespace", input: "  \t\n", want: ""},
		{name: "valid JSON", input: ` {"type":"object"} `, want: `{"type":"object"}`},
		{name: "invalid JSON", input: "{", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeOptionalParameterSchema(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeOptionalParameterSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("normalizeOptionalParameterSchema() = %q, want %q", got, tt.want)
			}
		})
	}
}
