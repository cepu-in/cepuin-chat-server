package chat

import "net/http"

func RegisterRoutes(
	mux *http.ServeMux,
	controller *Controller,
) {
	mux.HandleFunc(
		"/chat/list",
		controller.GetList,
	)

	mux.HandleFunc(
		"/chat/history",
		controller.GetHistory,
	)
}
