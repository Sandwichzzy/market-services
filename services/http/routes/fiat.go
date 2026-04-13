package routes

import (
	"fmt"
	"net/http"

	"github.com/Sandwichzzy/market-services/services/http/model"
)

func (h Routes) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	var req model.ListCurrenciesRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	resp, err := h.srv.ListCurrencies(&req)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		return
	}
	if err = jsonResponse(w, resp, http.StatusOK); err != nil {
		fmt.Println("Error writing response", "err", err.Error())
	}
}

func (h Routes) GetSymbolMarketCurrency(w http.ResponseWriter, r *http.Request) {
	var req model.GetSymbolMarketCurrencyRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	resp, err := h.srv.GetSymbolMarketCurrency(&req)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		return
	}
	if err = jsonResponse(w, resp, http.StatusOK); err != nil {
		fmt.Println("Error writing response", "err", err.Error())
	}
}
