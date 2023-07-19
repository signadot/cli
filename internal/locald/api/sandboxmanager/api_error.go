package sandboxmanager

func APIErrorResponse(err error) *ApplySandboxResponse {
	return &ApplySandboxResponse{
		It: &ApplySandboxResponse_ApiError{
			ApiError: err.Error(),
		},
	}
}
