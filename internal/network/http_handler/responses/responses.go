package responses

import (
	"encoding/json"
	"net/http"
)

const ResOk int = 1000
const ResAuthFailed = 1001
const ResGenTokenFailed = 1002
const ResInvalidSignature = 1003
const ResParseTokenFailed = 1004
const ResInvalidToken = 1005
const ResTokenExpired = 1006
const ResRetrieveFailed = 1007

type ResponseForm struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func CustomResponse(w http.ResponseWriter, code int, message string, data interface{}) error {
	w.Header().Add("Content-Type", "application/json")
	res := ResponseForm{
		Code:    code,
		Message: message,
		Data:    data,
	}
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(res)
}
