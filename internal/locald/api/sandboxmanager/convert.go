package sandboxmanager

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/signadot/cli/internal/config"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

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

func StatusToMap(status *StatusResponse) (map[string]any, error) {
	statusBytes, err := (protojson.MarshalOptions{}).Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("couldn't marshal status, %v", err)
	}
	d := json.NewDecoder(bytes.NewReader(statusBytes))
	d.UseNumber()
	var statusMap map[string]any
	if err := d.Decode(&statusMap); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal status, %v", err)
	}
	return statusMap, nil
}
