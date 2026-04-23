package copilot

import "testing"

func TestParseBaseURL(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]any
		want string
	}{
		{
			name: "proxy endpoint",
			in: map[string]any{
				"proxy_endpoint": "https://proxy.business.githubcopilot.com",
			},
			want: "https://api.business.githubcopilot.com",
		},
		{
			name: "proxy ep short key",
			in: map[string]any{
				"proxy_ep": "https://proxy.enterprise.githubcopilot.com",
			},
			want: "https://api.enterprise.githubcopilot.com",
		},
		{
			name: "endpoints api",
			in: map[string]any{
				"endpoints": map[string]any{
					"api": "https://api.githubcopilot.com",
				},
			},
			want: "https://api.githubcopilot.com",
		},
	}
	for _, tc := range cases {
		got := parseBaseURL(tc.in)
		if got != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
