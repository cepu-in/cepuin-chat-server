package websocket

import "net/http"

func RegisterRoutes(
	mux *http.ServeMux,
	controller *Controller,
) {
	mux.HandleFunc(
		"/ws",
		controller.Handle,
	)
}
