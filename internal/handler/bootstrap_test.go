package handler

import (
	"context"
	"testing"
)

func TestHandleBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request BootstrapRequest
		want    BootstrapResponse
	}{
		{
			name:    "uses the provided name",
			request: BootstrapRequest{Name: "developer"},
			want:    BootstrapResponse{Message: "hello, developer"},
		},
		{
			name:    "uses a default when name is empty",
			request: BootstrapRequest{},
			want:    BootstrapResponse{Message: "hello, world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := HandleBootstrap(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("HandleBootstrap() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("HandleBootstrap() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
