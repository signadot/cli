package hack

import (
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// SendEmptyBody is an SDK ClientOption that forces the request to send an empty
// body ("{}") instead of no body at all.
//
// TODO: Fix the underlying SDK instead.
func SendEmptyBody(op *runtime.ClientOperation) {
	op.Params = sendEmptyBody{delegate: op.Params}
}

type sendEmptyBody struct {
	delegate runtime.ClientRequestWriter
}

func (s sendEmptyBody) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {
	if err := s.delegate.WriteToRequest(r, reg); err != nil {
		return err
	}
	// Force the SDK to send a POST body.
	return r.SetBodyParam(struct{}{})
}
