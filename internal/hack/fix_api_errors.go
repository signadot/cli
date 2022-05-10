package hack

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
)

// FixAPIErrors is middleware that extracts the remote, server-provided API
// error message from an SDK error.
//
// TODO: Fix the SDK itself to return useful errors.
func FixAPIErrors(transport runtime.ClientTransport) runtime.ClientTransport {
	return apiErrorTransport{base: transport}
}

type apiErrorTransport struct {
	base runtime.ClientTransport
}

func (t apiErrorTransport) Submit(op *runtime.ClientOperation) (interface{}, error) {
	op.Reader = apiErrorReader{base: op.Reader}
	return t.base.Submit(op)
}

type apiErrorReader struct {
	base runtime.ClientResponseReader
}

func (r apiErrorReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	code := response.Code()
	switch {
	case code >= 400 && code <= 599:
		var apiErr apiError
		if err := consumer.Consume(response.Body(), &apiErr); err != nil && err != io.EOF {
			return nil, fmt.Errorf("can't read response body: %w", err)
		}
		return nil, fmt.Errorf("%v: %v", response.Message(), apiErr.Error)
	default:
		return r.base.ReadResponse(response, consumer)
	}
}

type apiError struct {
	Error     string `json:"error"`
	RequestID string `json:"requestID"`
}
