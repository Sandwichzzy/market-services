package dex

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Sandwichzzy/market-services/common/tasks"
	"github.com/Sandwichzzy/market-services/config"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// Uniswap V2 pair合约接口
const (
	pairABI = `[
		{
			"constant": true,
			"inputs": [],
			"name": "getReserves",
			"outputs": [
				{"internalType": "uint112","name": "_reserve0","type": "uint112"},
				{"internalType": "uint112","name": "_reserve1","type": "uint112"},
				{"internalType": "uint32","name": "_blockTimestampLast","type": "uint32"}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [],
			"name": "token0",
			"outputs": [{"internalType": "address","name":"","type":"address"}],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [],
			"name": "token1",
			"outputs": [{"internalType": "address","name":"","type":"address"}],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		}
	]`

	stakingRewardsABI = `[
		{
			"constant": true,
			"inputs": [],
			"name": "stakingToken",
			"outputs": [{"internalType": "address","name":"","type":"address"}],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		}
	]`

	erc20ABI = `[
		{
			"constant": true,
			"inputs": [],
			"name": "decimals",
			"outputs": [{"internalType": "uint8","name":"","type":"uint8"}],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [],
			"name": "symbol",
			"outputs": [{"internalType": "string","name":"","type":"string"}],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		}
	]`

	contractCallTimeout = 10 * time.Second
)

type Reserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

type pairMetadata struct {
	PairAddress    common.Address
	Token0         common.Address
	Token1         common.Address
	BaseToken      common.Address
	QuoteToken     common.Address
	Token0Symbol   string
	Token1Symbol   string
	BaseSymbol     string
	QuoteSymbol    string
	Token0Decimals int
	Token1Decimals int
	BaseDecimals   int
	QuoteDecimals  int
}

type DexMarketPrice struct {
	ChainClient       *ethclient.Client
	ChainID           *big.Int
	RouteAddress      common.Address
	QuoteTokenAddress common.Address
	TokenPair         common.Address
	LoopInterval      time.Duration
	pairABI           abi.ABI
	stakingRewardsABI abi.ABI
	erc20ABI          abi.ABI
	pair              *pairMetadata
	resourceCtx       context.Context
	resourceCancel    context.CancelFunc
	tasks             tasks.Group
	stopped           atomic.Bool
}

func NewDexMarketPrice(cfg *config.Config, shutdown context.CancelCauseFunc) (*DexMarketPrice, error) {
	if !common.IsHexAddress(cfg.Chain.RouteAddress) {
		return nil, fmt.Errorf("invalid route address: %s", cfg.Chain.RouteAddress)
	}
	if !common.IsHexAddress(cfg.Chain.QuoteTokenAddress) {
		return nil, fmt.Errorf("invalid quote token address: %s", cfg.Chain.QuoteTokenAddress)
	}
	if !common.IsHexAddress(cfg.Chain.TokenPair) {
		return nil, fmt.Errorf("invalid token pair address: %s", cfg.Chain.TokenPair)
	}

	dialCtx, dialCancel := context.WithTimeout(context.Background(), contractCallTimeout)
	defer dialCancel()
	chainClient, err := ethclient.DialContext(dialCtx, cfg.Chain.ChainRpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial chain client: %s", err)
	}
	log.Info("Contract caller client  init successfully")

	callCtx, cancel := context.WithTimeout(context.Background(), contractCallTimeout)
	defer cancel()
	chainIDBig, err := chainClient.ChainID(callCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to query chain id: %s", err)
	}
	parsedPairABI, err := abi.JSON(strings.NewReader(pairABI))
	if err != nil {
		return nil, fmt.Errorf("parse pair abi: %w", err)
	}
	parsedStakingRewardsABI, err := abi.JSON(strings.NewReader(stakingRewardsABI))
	if err != nil {
		return nil, fmt.Errorf("parse staking rewards abi: %w", err)
	}
	parsedERC20ABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("parse erc20 abi: %w", err)
	}
	resourceCtx, resourceCancel := context.WithCancel(context.Background())
	return &DexMarketPrice{
		ChainClient:       chainClient,
		ChainID:           chainIDBig,
		RouteAddress:      common.HexToAddress(cfg.Chain.RouteAddress),
		QuoteTokenAddress: common.HexToAddress(cfg.Chain.QuoteTokenAddress),
		TokenPair:         common.HexToAddress(cfg.Chain.TokenPair),
		LoopInterval:      time.Second * 5,
		pairABI:           parsedPairABI,
		stakingRewardsABI: parsedStakingRewardsABI,
		erc20ABI:          parsedERC20ABI,
		resourceCtx:       resourceCtx,
		resourceCancel:    resourceCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutdown(fmt.Errorf("dex crawler critical error %v", err))
			},
		},
	}, nil
}

func (dmp *DexMarketPrice) Start(ctx context.Context) error {
	dmp.tasks.Go(func() error {
		tickerOperator := time.NewTicker(dmp.LoopInterval)
		defer tickerOperator.Stop()

		if err := dmp.syncMarketPrice(); err != nil {
			dmp.tasks.HandleCrit(fmt.Errorf("failed to sync dex market price: %v", err))
			return err
		}

		for {
			select {
			case <-tickerOperator.C:
				if err := dmp.syncMarketPrice(); err != nil {
					dmp.tasks.HandleCrit(fmt.Errorf("failed to sync dex market price: %v", err))
					return err
				}
			case <-dmp.resourceCtx.Done():
				log.Info("get reserves shutting down")
				return nil
			}
		}
	})
	return nil
}

func (dmp *DexMarketPrice) Stop(ctx context.Context) error {
	dmp.resourceCancel()
	err := dmp.tasks.Wait()
	dmp.ChainClient.Close()
	dmp.stopped.Store(true)
	return err
}

func (dmp *DexMarketPrice) Stopped() bool {
	return dmp.stopped.Load()
}

func (dmp *DexMarketPrice) syncMarketPrice() error {
	log.Info("Fetching dex market price start", "pair", dmp.TokenPair.Hex())
	marketPrice, err := dmp.FetchMarketPrice()
	if err != nil {
		log.Error("FetchMarketPrice fail", "pair", dmp.TokenPair.Hex(), "error", err)
		return err
	}
	log.Info("FetchMarketPrice finished",
		"pair", dmp.pair.PairAddress.Hex(),
		"token0", dmp.pair.Token0.Hex(),
		"token0Symbol", dmp.pair.Token0Symbol,
		"token0Decimals", dmp.pair.Token0Decimals,
		"token1", dmp.pair.Token1.Hex(),
		"token1Symbol", dmp.pair.Token1Symbol,
		"token1Decimals", dmp.pair.Token1Decimals,
		"baseToken", dmp.pair.BaseToken.Hex(),
		"baseSymbol", dmp.pair.BaseSymbol,
		"quoteToken", dmp.pair.QuoteToken.Hex(),
		"quoteSymbol", dmp.pair.QuoteSymbol,
		"marketPrice", marketPrice.Text('f', 18),
	)
	return nil
}

func (dmp *DexMarketPrice) FetchMarketPrice() (*big.Float, error) {
	ctx, cancel := context.WithTimeout(context.Background(), contractCallTimeout)
	defer cancel()

	pair, err := dmp.ensurePairMetadata(ctx)
	if err != nil {
		return nil, err
	}

	reserves, err := dmp.fetchReserves(ctx, pair.PairAddress)
	if err != nil {
		return nil, err
	}
	return CalcPrice(pair, reserves)
}

func (dmp *DexMarketPrice) ensurePairMetadata(ctx context.Context) (*pairMetadata, error) {
	if dmp.pair != nil {
		return dmp.pair, nil
	}

	pair, err := dmp.fetchPairMetadata(ctx, dmp.TokenPair)
	if err == nil {
		dmp.pair = pair
		return pair, nil
	}

	stakingContract := dmp.TokenPair
	stakingToken, err := dmp.fetchStakingToken(ctx, stakingContract)
	if err != nil {
		return nil, fmt.Errorf("token-pair is neither an LP pair nor a staking rewards contract: %w", err)
	}

	pair, err = dmp.fetchPairMetadata(ctx, stakingToken)
	if err != nil {
		return nil, fmt.Errorf("staking token %s is not a valid LP pair: %w", stakingToken.Hex(), err)
	}

	dmp.TokenPair = stakingToken
	dmp.pair = pair
	log.Info("Resolved staking rewards contract to LP pair", "stakingContract", stakingContract.Hex(), "pair", stakingToken.Hex())
	return pair, nil
}

func (dmp *DexMarketPrice) fetchPairMetadata(ctx context.Context, pairAddress common.Address) (*pairMetadata, error) {
	token0, err := dmp.fetchAddress(ctx, pairAddress, dmp.pairABI, "token0")
	if err != nil {
		return nil, fmt.Errorf("token0 call failed on %s: %w", pairAddress.Hex(), err)
	}
	token1, err := dmp.fetchAddress(ctx, pairAddress, dmp.pairABI, "token1")
	if err != nil {
		return nil, fmt.Errorf("token1 call failed on %s: %w", pairAddress.Hex(), err)
	}

	token0Decimals, err := dmp.fetchDecimals(ctx, token0)
	if err != nil {
		return nil, fmt.Errorf("token0 decimals call failed on %s: %w", token0.Hex(), err)
	}
	token1Decimals, err := dmp.fetchDecimals(ctx, token1)
	if err != nil {
		return nil, fmt.Errorf("token1 decimals call failed on %s: %w", token1.Hex(), err)
	}

	token0Symbol := dmp.fetchSymbolOrAddress(ctx, token0)
	token1Symbol := dmp.fetchSymbolOrAddress(ctx, token1)
	quoteToken := dmp.QuoteTokenAddress
	pair := &pairMetadata{
		PairAddress:    pairAddress,
		Token0:         token0,
		Token1:         token1,
		QuoteToken:     quoteToken,
		Token0Symbol:   token0Symbol,
		Token1Symbol:   token1Symbol,
		Token0Decimals: token0Decimals,
		Token1Decimals: token1Decimals,
	}

	switch {
	case token0 == quoteToken:
		pair.BaseToken = token1
		pair.BaseSymbol = token1Symbol
		pair.BaseDecimals = token1Decimals
		pair.QuoteSymbol = token0Symbol
		pair.QuoteDecimals = token0Decimals
	case token1 == quoteToken:
		pair.BaseToken = token0
		pair.BaseSymbol = token0Symbol
		pair.BaseDecimals = token0Decimals
		pair.QuoteSymbol = token1Symbol
		pair.QuoteDecimals = token1Decimals
	default:
		return nil, fmt.Errorf("quote token %s is not in pair %s: token0=%s token1=%s",
			quoteToken.Hex(), pairAddress.Hex(), token0.Hex(), token1.Hex())
	}

	return pair, nil
}

func (dmp *DexMarketPrice) fetchStakingToken(ctx context.Context, stakingContract common.Address) (common.Address, error) {
	return dmp.fetchAddress(ctx, stakingContract, dmp.stakingRewardsABI, "stakingToken")
}

func (dmp *DexMarketPrice) fetchAddress(ctx context.Context, contract common.Address, parsedABI abi.ABI, method string) (common.Address, error) {
	output, err := dmp.callContractMethod(ctx, contract, parsedABI, method)
	if err != nil {
		return common.Address{}, err
	}

	values, err := parsedABI.Unpack(method, output)
	if err != nil {
		return common.Address{}, err
	}
	if len(values) != 1 {
		return common.Address{}, fmt.Errorf("unexpected %s output length: %d", method, len(values))
	}

	address, ok := values[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("unexpected %s output type: %T", method, values[0])
	}
	return address, nil
}

func (dmp *DexMarketPrice) fetchDecimals(ctx context.Context, token common.Address) (int, error) {
	output, err := dmp.callContractMethod(ctx, token, dmp.erc20ABI, "decimals")
	if err != nil {
		return 0, err
	}

	values, err := dmp.erc20ABI.Unpack("decimals", output)
	if err != nil {
		return 0, err
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("unexpected decimals output length: %d", len(values))
	}

	decimals, ok := values[0].(uint8)
	if !ok {
		return 0, fmt.Errorf("unexpected decimals output type: %T", values[0])
	}
	return int(decimals), nil
}

func (dmp *DexMarketPrice) fetchSymbolOrAddress(ctx context.Context, token common.Address) string {
	output, err := dmp.callContractMethod(ctx, token, dmp.erc20ABI, "symbol")
	if err != nil {
		log.Warn("symbol call failed", "token", token.Hex(), "error", err)
		return token.Hex()
	}

	values, err := dmp.erc20ABI.Unpack("symbol", output)
	if err != nil || len(values) != 1 {
		log.Warn("symbol unpack failed", "token", token.Hex(), "error", err)
		return token.Hex()
	}

	symbol, ok := values[0].(string)
	if !ok || symbol == "" {
		return token.Hex()
	}
	return symbol
}

func (dmp *DexMarketPrice) callContractMethod(ctx context.Context, contract common.Address, parsedABI abi.ABI, method string) ([]byte, error) {
	data, err := parsedABI.Pack(method)
	if err != nil {
		return nil, err
	}

	output, err := dmp.ChainClient.CallContract(ctx, ethereumCallMsg(contract, data), nil)
	if err != nil {
		return nil, fmt.Errorf("%s call failed on %s: %w", method, contract.Hex(), err)
	}
	return output, nil
}

func (dmp *DexMarketPrice) fetchReserves(ctx context.Context, pairAddress common.Address) (*Reserves, error) {
	output, err := dmp.callContractMethod(ctx, pairAddress, dmp.pairABI, "getReserves")
	if err != nil {
		return nil, err
	}
	values, err := dmp.pairABI.Unpack("getReserves", output)
	if err != nil {
		return nil, err
	}
	if len(values) != 3 {
		return nil, fmt.Errorf("unexpected getReserves output length: %d", len(values))
	}

	reserve0, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected reserve0 output type: %T", values[0])
	}
	reserve1, ok := values[1].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected reserve1 output type: %T", values[1])
	}
	blockTimestampLast, ok := values[2].(uint32)
	if !ok {
		return nil, fmt.Errorf("unexpected block timestamp output type: %T", values[2])
	}

	return &Reserves{
		Reserve0:           reserve0,
		Reserve1:           reserve1,
		BlockTimestampLast: blockTimestampLast,
	}, nil
}

func ethereumCallMsg(to common.Address, data []byte) ethereum.CallMsg {
	return ethereum.CallMsg{
		To:   &to,
		Data: data,
	}
}

func CalcPrice(pair *pairMetadata, reserves *Reserves) (*big.Float, error) {
	if pair == nil {
		return nil, fmt.Errorf("pair metadata is required")
	}
	if reserves == nil || reserves.Reserve0 == nil || reserves.Reserve1 == nil {
		return nil, fmt.Errorf("reserves are required")
	}

	reserve0 := scaleReserve(reserves.Reserve0, pair.Token0Decimals)
	reserve1 := scaleReserve(reserves.Reserve1, pair.Token1Decimals)

	switch {
	case pair.BaseToken == pair.Token0 && pair.QuoteToken == pair.Token1:
		return divideReserve(reserve1, reserve0)
	case pair.BaseToken == pair.Token1 && pair.QuoteToken == pair.Token0:
		return divideReserve(reserve0, reserve1)
	default:
		return nil, fmt.Errorf("pair metadata does not match price direction")
	}
}

func scaleReserve(reserve *big.Int, decimals int) *big.Float {
	value := new(big.Float).SetPrec(256).SetInt(reserve)
	scale := new(big.Float).SetPrec(256).SetInt(pow10(decimals))
	return value.Quo(value, scale)
}

func divideReserve(numerator *big.Float, denominator *big.Float) (*big.Float, error) {
	if denominator.Sign() == 0 {
		return nil, fmt.Errorf("denominator reserve is zero")
	}
	return new(big.Float).SetPrec(256).Quo(numerator, denominator), nil
}

func pow10(decimals int) *big.Int {
	if decimals <= 0 {
		return big.NewInt(1)
	}
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
}
