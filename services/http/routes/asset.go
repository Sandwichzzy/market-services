package routes

import (
	"fmt"
	"net/http"

	"github.com/Sandwichzzy/market-services/services/http/model"
	"github.com/ethereum/go-ethereum/log"
)

func (h Routes) GetSupportAssets(w http.ResponseWriter, r *http.Request) {
	var saReq model.SupportAssetRequest
	if err := decodeJSON(r, &saReq); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	log.Info("decode params success", "ConsumerToken", saReq.ConsumerToken)
	supReturn, err := h.srv.GetSupportAsset(&saReq)
	if err != nil {
		return
	}
	err = jsonResponse(w, supReturn, http.StatusOK)
	if err != nil {
		fmt.Println("Error writing response", "err", err.Error())
	}
}
