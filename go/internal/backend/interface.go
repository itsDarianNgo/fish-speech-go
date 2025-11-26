package backend

import (
	"context"
	"io"

	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// Backend defines the interface for communicating with the Python server.
type Backend interface {
	Health(ctx context.Context) error
	TTS(ctx context.Context, req *schema.ServeTTSRequest) ([]byte, string, error)
	TTSStream(ctx context.Context, req *schema.ServeTTSRequest) (io.ReadCloser, error)
	VQGANEncode(ctx context.Context, req *schema.ServeVQGANEncodeRequest) (*schema.ServeVQGANEncodeResponse, error)
	VQGANDecode(ctx context.Context, req *schema.ServeVQGANDecodeRequest) (*schema.ServeVQGANDecodeResponse, error)
	AddReference(ctx context.Context, req *schema.AddReferenceRequest) (*schema.AddReferenceResponse, error)
	ListReferences(ctx context.Context) (*schema.ListReferencesResponse, error)
	DeleteReference(ctx context.Context, id string) (*schema.DeleteReferenceResponse, error)
}

// Ensure BackendClient implements Backend.
var _ Backend = (*BackendClient)(nil)
