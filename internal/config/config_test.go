package config

import "testing"

func TestValidateBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "https host", in: "https://api.example.com", want: "https://api.example.com"},
		{name: "trailing slash trimmed", in: "https://api.example.com/", want: "https://api.example.com"},
		{name: "path kept", in: "https://api.example.com/base/", want: "https://api.example.com/base"},
		{name: "http localhost", in: "http://localhost:8080", want: "http://localhost:8080"},
		{name: "http loopback ip", in: "http://127.0.0.1:8080", want: "http://127.0.0.1:8080"},
		{name: "http remote rejected", in: "http://api.example.com", wantErr: true},
		{name: "credentials rejected", in: "https://user:pw@api.example.com", wantErr: true},
		{name: "query rejected", in: "https://api.example.com?x=1", wantErr: true},
		{name: "no scheme rejected", in: "api.example.com", wantErr: true},
		{name: "garbage rejected", in: "::not a url", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateBaseURL(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("validateBaseURL(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateBaseURL(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("validateBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
