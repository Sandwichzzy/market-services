package routes

import (
	"fmt"
	"net/http"

	"github.com/Sandwichzzy/market-services/services/http/model"
)

func (h Routes) ListKlines(w http.ResponseWriter, r *http.Request) {
	var req model.ListKlinesRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	resp, err := h.srv.ListKlines(&req)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		return
	}
	if err = jsonResponse(w, resp, http.StatusOK); err != nil {
		fmt.Println("Error writing response", "err", err.Error())
	}
}

func (h Routes) ListExchangeKlines(w http.ResponseWriter, r *http.Request) {
	var req model.ListExchangeKlinesRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	resp, err := h.srv.ListExchangeKlines(&req)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		return
	}
	if err = jsonResponse(w, resp, http.StatusOK); err != nil {
		fmt.Println("Error writing response", "err", err.Error())
	}
}
