package auth

import (
	"encoding/json"
	"promoter/pkg/tg"
	"promoter/pkg/tg/tdjson"
	"unsafe"
)

//GetAuthorizationState GetAuthorizationState()
func GetAuthorizationState(client unsafe.Pointer) (tdjson.GetAuthorizationStateResponse, error) {
	var b, _ = json.Marshal(tdjson.TDCommon{Type: "getAuthorizationState"})

	tg.Send(client, string(b))
	var ret tdjson.GetAuthorizationStateResponse
	if err := json.Unmarshal([]byte(tg.Receive(client)), &ret); err != nil {
		return ret, err
	}

	return ret, nil
}
