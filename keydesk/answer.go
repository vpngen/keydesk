package keydesk

import "github.com/vpngen/keydesk/gen/models"

const (
	AnswerStatusSuccess = "success"
	AnswerStatusError   = "error"
)

// Answer - answer from keydesk
// Means: HTTP/1.1 Code Desc
// Code: 200, 400, 500
// Desc: OK, Bad Request, Internal Server Error
// Status: 'success' or 'error'
// Message: error message
type Answer struct {
	Code    int            `json:"code"`
	Desc    string         `json:"desc"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Configs models.Newuser `json:"configs"`
}
