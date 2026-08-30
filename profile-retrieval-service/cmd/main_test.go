package main

import "testing"

func TestListenAddrFromEnv(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{
			name: "default",
			want: ":8080",
		},
		{
			name: "port without colon",
			port: "10000",
			want: ":10000",
		},
		{
			name: "port with colon",
			port: ":10000",
			want: ":10000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", tt.port)

			if got := listenAddrFromEnv(); got != tt.want {
				t.Fatalf("listenAddrFromEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}
