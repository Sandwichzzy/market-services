package dex

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestCalcPrice_BaseToken0QuoteToken1(t *testing.T) {
	base := common.HexToAddress("0x0000000000000000000000000000000000000001")
	quote := common.HexToAddress("0x0000000000000000000000000000000000000002")
	pair := &pairMetadata{
		Token0:         base,
		Token1:         quote,
		BaseToken:      base,
		QuoteToken:     quote,
		Token0Decimals: 18,
		Token1Decimals: 6,
	}
	reserves := &Reserves{
		Reserve0: pow10(18),
		Reserve1: new(big.Int).Mul(big.NewInt(2000), pow10(6)),
	}

	price, err := CalcPrice(pair, reserves)
	if err != nil {
		t.Fatalf("CalcPrice returned error: %v", err)
	}
	if got := price.Text('f', 0); got != "2000" {
		t.Fatalf("price = %s, want 2000", got)
	}
}

func TestCalcPrice_BaseToken1QuoteToken0(t *testing.T) {
	base := common.HexToAddress("0x0000000000000000000000000000000000000001")
	quote := common.HexToAddress("0x0000000000000000000000000000000000000002")
	pair := &pairMetadata{
		Token0:         quote,
		Token1:         base,
		BaseToken:      base,
		QuoteToken:     quote,
		Token0Decimals: 6,
		Token1Decimals: 18,
	}
	reserves := &Reserves{
		Reserve0: new(big.Int).Mul(big.NewInt(2000), pow10(6)),
		Reserve1: pow10(18),
	}

	price, err := CalcPrice(pair, reserves)
	if err != nil {
		t.Fatalf("CalcPrice returned error: %v", err)
	}
	if got := price.Text('f', 0); got != "2000" {
		t.Fatalf("price = %s, want 2000", got)
	}
}

func TestCalcPrice_MixedDecimals(t *testing.T) {
	base := common.HexToAddress("0x0000000000000000000000000000000000000001")
	quote := common.HexToAddress("0x0000000000000000000000000000000000000002")
	pair := &pairMetadata{
		Token0:         base,
		Token1:         quote,
		BaseToken:      base,
		QuoteToken:     quote,
		Token0Decimals: 8,
		Token1Decimals: 18,
	}
	reserves := &Reserves{
		Reserve0: pow10(8),
		Reserve1: new(big.Int).Mul(big.NewInt(15), pow10(18)),
	}

	price, err := CalcPrice(pair, reserves)
	if err != nil {
		t.Fatalf("CalcPrice returned error: %v", err)
	}
	if got := price.Text('f', 0); got != "15" {
		t.Fatalf("price = %s, want 15", got)
	}
}

func TestCalcPrice_RejectsZeroBaseReserve(t *testing.T) {
	base := common.HexToAddress("0x0000000000000000000000000000000000000001")
	quote := common.HexToAddress("0x0000000000000000000000000000000000000002")
	pair := &pairMetadata{
		Token0:         base,
		Token1:         quote,
		BaseToken:      base,
		QuoteToken:     quote,
		Token0Decimals: 18,
		Token1Decimals: 18,
	}
	reserves := &Reserves{
		Reserve0: big.NewInt(0),
		Reserve1: pow10(18),
	}

	if _, err := CalcPrice(pair, reserves); err == nil {
		t.Fatal("CalcPrice should reject zero base reserve")
	}
}

func TestCalcPrice_RejectsMismatchedPriceDirection(t *testing.T) {
	token0 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	token1 := common.HexToAddress("0x0000000000000000000000000000000000000002")
	quote := common.HexToAddress("0x0000000000000000000000000000000000000003")
	pair := &pairMetadata{
		Token0:         token0,
		Token1:         token1,
		BaseToken:      token0,
		QuoteToken:     quote,
		Token0Decimals: 18,
		Token1Decimals: 18,
	}
	reserves := &Reserves{
		Reserve0: pow10(18),
		Reserve1: pow10(18),
	}

	if _, err := CalcPrice(pair, reserves); err == nil {
		t.Fatal("CalcPrice should reject quote tokens outside the pair")
	}
}
