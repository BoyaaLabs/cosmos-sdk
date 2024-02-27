package cometbft

import (
	"context"
	"fmt"

	"cosmossdk.io/core/transaction"
	"cosmossdk.io/log"
	"cosmossdk.io/runtime/v2"
	serverv2 "cosmossdk.io/server/v2"
	cometlog "cosmossdk.io/server/v2/cometbft/log"
	"cosmossdk.io/server/v2/cometbft/types"
	"cosmossdk.io/server/v2/mempool"
	abciserver "github.com/cometbft/cometbft/abci/server"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	cmttypes "github.com/cometbft/cometbft/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CometBFTServer[T transaction.Tx] struct {
	Node     *node.Node
	CometBFT *Consensus[T]
	logger   log.Logger

	config    Config
	cleanupFn func()
}

func NewCometBFTServer[T transaction.Tx](
	app *runtime.App,
	cfg Config,
	voteExtHandler types.VoteExtensionsHandler,
) *CometBFTServer[T] {
	logger := app.GetLogger().With("module", "cometbft-server")

	// create noop mempool
	mempool := mempool.NewNoopMempool[T]()

	// TODO: set default handlers (prepare, process, extendvote, verify vote)
	// TODO: set
	return &CometBFTServer[T]{
		logger:   logger,
		CometBFT: NewConsensus[T](app.AppManager[T], mempool, app.GetStore(), cfg),
		config:   cfg,
	}

}

func (s *CometBFTServer[T]) Name() string {
	return "cometbft"
}

func (s *CometBFTServer[T]) Start(ctx context.Context) error {
	wrappedLogger := cometlog.CometLoggerWrapper{Logger: s.logger}
	if s.config.Standalone {
		svr, err := abciserver.NewServer(s.config.Addr, s.config.Transport, s.CometBFT)
		if err != nil {
			return fmt.Errorf("error creating listener: %w", err)
		}

		svr.SetLogger(wrappedLogger)

		return svr.Start()
	}

	nodeKey, err := p2p.LoadOrGenNodeKey(s.config.CmtConfig.NodeKeyFile())
	if err != nil {
		return err
	}

	s.Node, err = node.NewNodeWithContext(
		ctx,
		s.config.CmtConfig,
		pvm.LoadOrGenFilePV(s.config.CmtConfig.PrivValidatorKeyFile(), s.config.CmtConfig.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(s.CometBFT),
		getGenDocProvider(s.config.CmtConfig),
		cmtcfg.DefaultDBProvider,
		node.DefaultMetricsProvider(s.config.CmtConfig.Instrumentation),
		wrappedLogger,
	)
	if err != nil {
		return err
	}

	s.cleanupFn = func() {
		if s.Node != nil && s.Node.IsRunning() {
			_ = s.Node.Stop()
		}
	}

	return s.Node.Start()
}

func (s *CometBFTServer[T]) Stop(_ context.Context) error {
	defer s.cleanupFn()
	if s.Node != nil {
		return s.Node.Stop()
	}
	return nil
}

func (s *CometBFTServer[T]) Config() (any, *viper.Viper) {
	v := viper.New()
	v.SetConfigFile("???") // TODO: where do we set this
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.ReadInConfig()
	return nil, nil
}

// returns a function which returns the genesis doc from the genesis file.
func getGenDocProvider(cfg *cmtcfg.Config) func() (*cmttypes.GenesisDoc, error) {
	return func() (*cmttypes.GenesisDoc, error) {
		appGenesis, err := genutiltypes.AppGenesisFromFile(cfg.GenesisFile())
		if err != nil {
			return nil, err
		}

		return appGenesis.ToGenesisDoc()
	}
}

func (s *CometBFTServer[T]) CLICommands() serverv2.CLIConfig {
	return serverv2.CLIConfig{
		Command: []*cobra.Command{
			s.StatusCommand(),
			s.ShowNodeIDCmd(),
			s.ShowValidatorCmd(),
			s.ShowAddressCmd(),
			s.VersionCmd(),
			s.QueryBlockCmd(),
			s.QueryBlocksCmd(),
			s.QueryBlockResultsCmd(),
		},
	}
}

/*
GetStore from app
GetLogger from app

// Set on abci.go
func SetCodec? <- I think we can get this from app manager too. Is codec.Codec fine?

func SetExtendVoteExtension
func SetVerifyVoteExtension
func SetPrepareProposalHandler
func SetProcessProposalHandler
func SetSnapshotManager (?)

API routes
SetServer grpc
grpc gateway
streaming
telemetry
cli commands



*/
