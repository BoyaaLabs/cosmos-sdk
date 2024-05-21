/*
NOTE: Usage of x/params to manage parameters is deprecated in favor of x/gov
controlled execution of MsgUpdateParams messages. These types remains solely
for migration purposes and will be removed in a future release.
*/
package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeyMintDenom     = []byte("MintDenom")
	KeyHalfYear      = []byte("HalfYear")
	KeyBeginBlock    = []byte("BeginBlock")
	KeyBlocksPerYear = []byte("BlocksPerYear")
)

// Deprecated: ParamTable for minting module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// Implements params.ParamSet
//
// Deprecated.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMintDenom, &p.Coin.Denom, validateMintDenom),
		paramtypes.NewParamSetPair(KeyHalfYear, &p.HalfYear, validateUin64),
		paramtypes.NewParamSetPair(KeyBlocksPerYear, &p.BlocksPerYear, validateUin64),
		paramtypes.NewParamSetPair(KeyBeginBlock, &p.BeginBlock, validateUin64),
	}
}
