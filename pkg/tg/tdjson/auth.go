package tdjson

//GetAuthorizationStateResponse GetAuthorizationState response
type GetAuthorizationStateResponse struct {
	TDCommon
	Name  string                             `json:"name"`
	Value GetAuthorizationStateResponseValue `json:"value"`
}

//GetAuthorizationStateResponseValue GetAuthorizationState response value
type GetAuthorizationStateResponseValue struct {
	TDCommon
	Value string `json:"value"`
}
