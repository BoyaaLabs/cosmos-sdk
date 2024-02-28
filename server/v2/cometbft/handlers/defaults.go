package handlers

import (
	"context"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/gogoproto/proto" // TODO: use protov2

	consensusv1 "cosmossdk.io/api/cosmos/consensus/v1"
	"cosmossdk.io/core/transaction"
	"cosmossdk.io/server/v2/core/appmanager"

	"cosmossdk.io/server/v2/cometbft/mempool"
	"cosmossdk.io/server/v2/core/store"
)

type AppManager[T transaction.Tx] interface {
	ValidateTx(ctx context.Context, tx T) (appmanager.TxResult, error)
	Query(ctx context.Context, version uint64, request appmanager.Type) (response appmanager.Type, err error)
}

type DefaultProposalHandler[T transaction.Tx] struct {
	mempool    mempool.Mempool[T]
	txSelector TxSelector[T]
}

func NewDefaultProposalHandler[T transaction.Tx](mp mempool.Mempool[T]) *DefaultProposalHandler[T] {
	return &DefaultProposalHandler[T]{
		mempool:    mp,
		txSelector: NewDefaultTxSelector[T](),
	}
}

func (h *DefaultProposalHandler[T]) PrepareHandler() PrepareHandler[T] {
	return func(ctx context.Context, app AppManager[T], txs []T, req proto.Message) ([]T, error) {
		abciReq, ok := req.(*abci.RequestPrepareProposal)
		if !ok {
			return nil, fmt.Errorf("invalid request type: %T", req)
		}

		var maxBlockGas uint64

		res, err := app.Query(ctx, 0, &consensusv1.QueryParamsRequest{})
		if err != nil {
			return nil, err
		}

		paramsResp, ok := res.(*consensusv1.QueryParamsResponse)
		if !ok {
			return nil, fmt.Errorf("unexpected consensus params response type; expected: %T, got: %T", &consensusv1.QueryParamsResponse{}, res)
		}

		if b := paramsResp.GetParams().Block; b != nil {
			maxBlockGas = uint64(b.MaxGas)
		}

		defer h.txSelector.Clear()

		// TODO: can we assume nil mempool is NoOp?
		// If the mempool is nil or NoOp we simply return the transactions
		// requested from CometBFT, which, by default, should be in FIFO order.
		//
		// Note, we still need to ensure the transactions returned respect req.MaxTxBytes.
		_, isNoOp := h.mempool.(mempool.NoOpMempool[T])
		if h.mempool == nil || isNoOp {
			for _, tx := range txs {
				stop := h.txSelector.SelectTxForProposal(ctx, uint64(abciReq.MaxTxBytes), maxBlockGas, tx)
				if stop {
					break
				}
			}

			return h.txSelector.SelectedTxs(ctx), nil
		}

		iterator := h.mempool.Select(ctx, txs)
		for iterator != nil {
			memTx := iterator.Tx()

			// NOTE: Since transaction verification was already executed in CheckTx,
			// which calls mempool.Insert, in theory everything in the pool should be
			// valid. But some mempool implementations may insert invalid txs, so we
			// check again.
			_, err := app.ValidateTx(ctx, memTx)
			if err != nil {
				err := h.mempool.Remove([]T{memTx})
				if err != nil && !errors.Is(err, mempool.ErrTxNotFound) {
					return nil, err
				}
			} else {
				stop := h.txSelector.SelectTxForProposal(ctx, uint64(abciReq.MaxTxBytes), maxBlockGas, memTx)
				if stop {
					break
				}
			}

			iterator = iterator.Next()
		}

		return h.txSelector.SelectedTxs(ctx), nil
	}
}

func (h *DefaultProposalHandler[T]) ProcessHandler() ProcessHandler[T] {
	return func(ctx context.Context, app AppManager[T], txs []T, req proto.Message) error {
		// If the mempool is nil we simply return ACCEPT,
		// because PrepareProposal may have included txs that could fail verification.
		_, isNoOp := h.mempool.(mempool.NoOpMempool[T])
		if h.mempool == nil || isNoOp {
			return nil
		}

		// TODO: not using this request for now
		_, ok := req.(*abci.RequestProcessProposal)
		if !ok {
			return fmt.Errorf("invalid request type: %T", req)
		}

		res, err := app.Query(ctx, 0, &consensusv1.QueryParamsRequest{})
		if err != nil {
			return err
		}

		paramsResp, ok := res.(*consensusv1.QueryParamsResponse)
		if !ok {
			return fmt.Errorf("unexpected consensus params response type; expected: %T, got: %T", &consensusv1.QueryParamsResponse{}, res)
		}

		var maxBlockGas uint64
		if b := paramsResp.GetParams().Block; b != nil {
			maxBlockGas = uint64(b.MaxGas)
		}

		var totalTxGas uint64
		for _, tx := range txs {
			_, err := app.ValidateTx(ctx, tx)
			if err != nil {
				return fmt.Errorf("failed to validate tx: %w", err)
			}

			if maxBlockGas > 0 {
				totalTxGas += tx.GetGasLimit()
				if totalTxGas > maxBlockGas {
					return fmt.Errorf("total tx gas %d exceeds max block gas %d", totalTxGas, maxBlockGas)
				}
			}
		}

		return nil
	}
}

// NoOpPrepareProposal defines a no-op PrepareProposal handler. It will always
// return the transactions sent by the client's request.
func NoOpPrepareProposal[T transaction.Tx]() PrepareHandler[T] {
	return func(ctx context.Context, app AppManager[T], txs []T, req proto.Message) ([]T, error) {
		return txs, nil
	}
}

// NoOpProcessProposal defines a no-op ProcessProposal Handler. It will always
// return ACCEPT.
func NoOpProcessProposal[T transaction.Tx]() ProcessHandler[T] {
	return func(context.Context, AppManager[T], []T, proto.Message) error {
		return nil
	}
}

// NoOpExtendVote defines a no-op ExtendVote handler. It will always return an
// empty byte slice as the vote extension.
func NoOpExtendVote() ExtendVoteHandler {
	return func(context.Context, store.ReaderMap, *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		return &abci.ResponseExtendVote{VoteExtension: []byte{}}, nil
	}
}

// NoOpVerifyVoteExtensionHandler defines a no-op VerifyVoteExtension handler. It
// will always return an ACCEPT status with no error.
func NoOpVerifyVoteExtensionHandler() VerifyVoteExtensionhandler {
	return func(context.Context, store.ReaderMap, *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}
