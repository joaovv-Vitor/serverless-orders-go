package handler

import "context"

// BootstrapRequest is the small input contract used only to verify the
// Lambda bootstrap. The HTTP order contract will be introduced in phase 2.
type BootstrapRequest struct {
	Name string `json:"name"`
}

// BootstrapResponse is returned by the bootstrap Lambda.
type BootstrapResponse struct {
	Message string `json:"message"`
}

// HandleBootstrap contains the testable logic of the bootstrap Lambda.
func HandleBootstrap(_ context.Context, request BootstrapRequest) (BootstrapResponse, error) {
	name := request.Name
	if name == "" {
		name = "world"
	}

	return BootstrapResponse{Message: "hello, " + name}, nil
}
