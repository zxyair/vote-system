package server

import (
	"errors"
	"testing"

	"vote-system/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToStatusMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "not found", err: service.ErrNotFound, want: codes.NotFound},
		{name: "unauthenticated", err: service.ErrUnauthenticated, want: codes.Unauthenticated},
		{name: "forbidden", err: service.ErrForbidden, want: codes.PermissionDenied},
		{name: "conflict", err: service.ErrConflict, want: codes.AlreadyExists},
		{name: "invalid", err: service.ErrInvalid, want: codes.InvalidArgument},
		{name: "internal", err: errors.New("boom"), want: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := status.Code(toStatus(tt.err))
			if got != tt.want {
				t.Fatalf("status code = %v, want %v", got, tt.want)
			}
		})
	}
}
