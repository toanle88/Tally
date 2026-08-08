// Package httpapi exposes the generated transport boundary to the API composition root.
package httpapi

import "github.com/toanle88/Tally/internal/platform/httpapi/generated"

// Handler is the generated contract handler boundary. Capability modules will
// implement it when their delivery items add the corresponding behavior.
type Handler = generated.Handler

// CommandRequest is the generated common command transport type.
type CommandRequest = generated.CommandRequest
