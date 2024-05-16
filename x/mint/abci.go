package mint

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/mint/keeper"
	"github.com/cosmos/cosmos-sdk/x/mint/types"
)

// BeginBlocker mints new tokens for the previous block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper, ic types.InflationCalculationFn) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	params := k.GetParams(ctx)

	halvings := uint64(ctx.BlockHeight()) / (params.BlocksPerYear * 4)
	initialReward := 21e7 / (params.BlocksPerYear * 4)

	for i := 0; i < int(halvings); i++ {
		initialReward /= 2
	}

	transferCoin := sdk.NewCoin(params.MintDenom, sdk.NewInt(int64(initialReward)))
	transferCoins := sdk.NewCoins(transferCoin)

	if k.HasBalance(ctx, transferCoin) {
		err := k.AddCollectedFees(ctx, transferCoins)
		if err != nil {
			panic(err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMint,
			//sdk.NewAttribute(types.AttributeKeyBondedRatio, bondedRatio.String()),
			//sdk.NewAttribute(types.AttributeKeyInflation, minter.Inflation.String()),
			//sdk.NewAttribute(types.AttributeKeyAnnualProvisions, minter.AnnualProvisions.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, transferCoin.Amount.String()),
		),
	)
}
