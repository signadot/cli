package sandboxmanager

import (
	"encoding/json"

	"github.com/signadot/cli/internal/config"
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

func ToGRPCSandboxSpec(sbs *models.SandboxSpec) (*structpb.Struct, error) {
	d, _ := json.Marshal(sbs)
	un := map[string]any{}
	if err := json.Unmarshal(d, &un); err != nil {
		return nil, err
	}
	return structpb.NewStruct(un)
}

// TODO maybe use generics here?
func ToModelsSandbox(grpcSandbox *structpb.Struct) (*models.Sandbox, error) {
	d, e := grpcSandbox.MarshalJSON()
	if e != nil {
		return nil, e
	}
	sb := &models.Sandbox{}
	if err := json.Unmarshal(d, sb); err != nil {
		return nil, err
	}
	return sb, nil
}

func ToGRPCCIConfig(ciConfig *config.ConnectInvocationConfig) (*structpb.Struct, error) {
	d, _ := json.Marshal(ciConfig)
	un := map[string]any{}
	if err := json.Unmarshal(d, &un); err != nil {
		return nil, err
	}
	return structpb.NewStruct(un)
}

func ToCIConfig(grpcSpec *structpb.Struct) (*config.ConnectInvocationConfig, error) {
	d, e := grpcSpec.MarshalJSON()
	if e != nil {
		return nil, e
	}
	ciConfig := &config.ConnectInvocationConfig{}
	if err := json.Unmarshal(d, ciConfig); err != nil {
		return nil, err
	}
	return ciConfig, nil
}
