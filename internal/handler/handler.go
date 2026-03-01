package handlers

import (
	"net/http"
)

func MainPageHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Welcome to go-musthave-metrics!"))
}
