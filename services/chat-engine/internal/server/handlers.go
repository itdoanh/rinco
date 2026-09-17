package server

import (
	"net/http"

	"github.com/itdoanh/rinco/services/chat-engine/internal/connectrpc"
	"github.com/itdoanh/rinco/services/chat-engine/internal/httpapi"
)

// newRESTHandler returns the HTTP handler that serves /v1/* and /healthz.
func newRESTHandler(d Deps) http.Handler {
	return httpapi.New(d.Chat, d.Group, d.Conv, d.Presence).Routes()
}

// newRPCHandler returns the HTTP handler that serves the Connect-RPC
// procedures mounted under /rinco.chat.v1.*.
func newRPCHandler(d Deps) http.Handler {
	mux := http.NewServeMux()

	chatSvc := connectrpc.NewChatService(d.Chat, d.Presence, 5)
	groupSvc := connectrpc.NewGroupService(d.Group)
	convSvc := connectrpc.NewConversationService(d.Conv)

	connectrpc.MountRoutes(mux, chatSvc, groupSvc, convSvc)
	return mux
}
