package mint

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mint/keeper"
	"github.com/cosmos/cosmos-sdk/x/mint/types"
)

// BeginBlocker mints new tokens for the previous block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	params := k.GetParams(ctx)
	transferCoin := calcOutCoin(params, ctx.BlockHeight())

	if transferCoin.Amount.GT(sdk.NewInt(0)) && k.GetBalance(ctx, params.Coin.Denom).Amount.GTE(transferCoin.Amount) {
		err := k.AddCollectedFees(ctx, sdk.NewCoins(transferCoin))
		if err != nil {
			panic(err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMint,
			sdk.NewAttribute(sdk.AttributeKeyAmount, transferCoin.Amount.String()),
		),
	)
}

func calcOutCoin(params types.Params, height int64) sdk.Coin {
	if uint64(height) < params.BeginBlock {
		return sdk.Coin{Denom: params.Coin.Denom, Amount: sdk.NewInt(0)}
	}
	fixedHeight := uint64(height) - params.BeginBlock
	halvings := fixedHeight / (params.BlocksPerYear * params.HalfYear)
	initialReward := params.Coin.Amount.Quo(sdk.NewInt(int64(params.BlocksPerYear * params.HalfYear)))

	for i := 0; i <= int(halvings); i++ {
		initialReward = initialReward.Quo(sdk.NewInt(2))
	}
	return sdk.NewCoin(params.Coin.Denom, initialReward)
}
