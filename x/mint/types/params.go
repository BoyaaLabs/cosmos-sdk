package types

import (
	"errors"
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewParams returns Params instance with the given values.
func NewParams(mintDenom string, mintAmount, blocksPerYear, halfYear, beginBlock uint64) Params {
	return Params{
		Coin:          sdk.NewCoin(mintDenom, sdk.NewInt(int64(mintAmount))),
		HalfYear:      halfYear,
		BeginBlock:    beginBlock,
		BlocksPerYear: blocksPerYear,
	}
}

// DefaultParams returns default x/mint module parameters.
func DefaultParams() Params {
	return Params{
		Coin:          sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(21e7)),
		HalfYear:      uint64(4),
		BeginBlock:    uint64(10000),
		BlocksPerYear: uint64(60 * 60 * 8766 / 5), // assuming 5 second block times
	}
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if err := sdk.ValidateDenom(p.Coin.Denom); err != nil {
		return err
	}
	if err := validateUin64(p.BlocksPerYear); err != nil {
		return err
	}
	if err := validateUin64(p.HalfYear); err != nil {
		return err
	}
	if err := validateUin64(p.BeginBlock); err != nil {
		return err
	}
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func validateMintDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if strings.TrimSpace(v) == "" {
		return errors.New("mint denom cannot be blank")
	}
	if err := sdk.ValidateDenom(v); err != nil {
		return err
	}

	return nil
}

func validateUin64(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("blocks per year must be positive: %d", v)
	}

	return nil
}
