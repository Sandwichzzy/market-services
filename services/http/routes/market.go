package routes

import (
	"fmt"
	"net/http"

	"github.com/Sandwichzzy/market-services/services/http/model"
)

func (h Routes) ListMarketSymbols(w http.ResponseWriter, r *http.Request) {
	var req model.ListMarketSymbolsRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	resp, err := h.srv.ListMarketSymbols(&req)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		return
	}
	if err = jsonResponse(w, resp, http.StatusOK); err != nil {
		fmt.Println("Error writing response", "err", err.Error())
	}
}

func (h Routes) ListSymbolMarkets(w http.ResponseWriter, r *http.Request) {
	var req model.ListSymbolMarketsRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	resp, err := h.srv.ListSymbolMarkets(&req)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		return
	}
	if err = jsonResponse(w, resp, http.StatusOK); err != nil {
		fmt.Println("Error writing response", "err", err.Error())
	}
}

func (h Routes) GetSymbolMarket(w http.ResponseWriter, r *http.Request) {
	var req model.GetSymbolMarketRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	resp, err := h.srv.GetSymbolMarket(&req)
	if err != nil {
		http.Error(w, InternalServerError, http.StatusInternalServerError)
		return
	}
	if err = jsonResponse(w, resp, http.StatusOK); err != nil {
		fmt.Println("Error writing response", "err", err.Error())
	}
}
