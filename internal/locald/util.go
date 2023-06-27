package locald

import (
	commonapi "github.com/signadot/cli/internal/locald/api"
	"github.com/signadot/libconnect/common/svchealth"
)

func toGRPCServiceHealth(csh *svchealth.ServiceHealth) *commonapi.ServiceHealth {
	return &commonapi.ServiceHealth{
		Healthy:         csh.Healthy,
		LastErrorReason: csh.LastErrorReason,
		ErrorCount:      csh.ErrorCount,
		LastErrorTime:   csh.LastErrorTime,
	}
}
