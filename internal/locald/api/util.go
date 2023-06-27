package api

import (
	"github.com/signadot/libconnect/common/svchealth"
)

func ToGRPCServiceHealth(csh *svchealth.ServiceHealth) *ServiceHealth {
	return &ServiceHealth{
		Healthy: csh.Healthy,
	}
}
