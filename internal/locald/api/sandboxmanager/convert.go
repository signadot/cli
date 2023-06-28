package sandboxmanager

import (
	"encoding/json"

	"github.com/signadot/go-sdk/models"
	"google.golang.org/protobuf/types/known/structpb"
)

func ToModelsSandboxSpec(grpcSpec *structpb.Struct) (*models.SandboxSpec, error) {
	d, e := grpcSpec.MarshalJSON()
	if e != nil {
		return nil, e
	}
	sbs := &models.SandboxSpec{}
	if err := json.Unmarshal(d, sbs); err != nil {
		return nil, err
	}
	return sbs, nil
}

func ToGRPCSandbox(sb *models.Sandbox) (*structpb.Struct, error) {
	d, _ := json.Marshal(sb)
	un := map[string]any{}
	if err := json.Unmarshal(d, &un); err != nil {
		return nil, err
	}
	return structpb.NewStruct(un)
}
