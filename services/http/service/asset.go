package service

import (
	"fmt"

	"github.com/Sandwichzzy/market-services/services/http/model"
	"github.com/ethereum/go-ethereum/log"
)

func (h HandleSvc) GetSupportAsset(request *model.SupportAssetRequest) (*model.SupportAssetResponse, error) {
	tokenReq := request.ConsumerToken
	assetList, err := h.assetView.QueryAssets()
	if err != nil {
		log.Error("query assets error", "error", err)
		return nil, err
	}
	var supportAssetList []model.SupportAsset
	for _, asset := range assetList {
		supportAsset := model.SupportAsset{
			Guid:        asset.Guid,
			AssetName:   asset.AssetName,
			AssetSymbol: asset.AssetSymbol,
			AssetLogo:   asset.AssetLogo,
		}
		supportAssetList = append(supportAssetList, supportAsset)
	}
	return &model.SupportAssetResponse{
		Code:    200,
		Message: fmt.Sprintf("here is support asset list, your query token=%s", tokenReq),
		Result:  supportAssetList,
	}, nil
}
