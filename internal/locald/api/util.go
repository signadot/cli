package api

import (
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/signadot/libconnect/common/svchealth"
)

func ToGRPCServiceHealth(csh *svchealth.ServiceHealth) *ServiceHealth {
	return &ServiceHealth{
		Healthy:         csh.Healthy,
		ErrorCount:      uint32(csh.ErrorCount),
		LastErrorReason: csh.LastErrorReason,
		LastErrorTime: &timestamp.Timestamp{
			Seconds: csh.LastErrorTime.Unix(),
		},
	}
}
