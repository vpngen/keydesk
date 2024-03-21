// Package messages provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.1.0 DO NOT EDIT.
package messages

import (
	"time"
)

const (
	JWTAuthScopes = "JWTAuth.Scopes"
)

// CreateMessageRequest defines model for CreateMessageRequest.
type CreateMessageRequest struct {
	Priority *int    `json:"priority,omitempty"`
	Text     string  `json:"text"`
	Ttl      *string `json:"ttl,omitempty"`
}

// Error defines model for Error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Message defines model for Message.
type Message struct {
	Id       int       `json:"id"`
	IsRead   bool      `json:"is_read"`
	Priority int       `json:"priority"`
	Text     string    `json:"text"`
	Time     time.Time `json:"time"`
	Ttl      string    `json:"ttl"`
}

// PostMessagesJSONRequestBody defines body for PostMessages for application/json ContentType.
type PostMessagesJSONRequestBody = CreateMessageRequest
