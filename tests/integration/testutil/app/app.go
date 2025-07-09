package app

import (
	"io"
	"os"

	"github.com/spf13/cast"

	dbm "github.com/cosmos/cosmos-db"
	evmante "github.com/cosmos/evm/ante"
	cosmosevmante "github.com/cosmos/evm/ante/evm"
	"github.com/cosmos/evm/app"
	evmconfig "github.com/cosmos/evm/config"
	srvflags "github.com/cosmos/evm/server/flags"
	cosmosevmtypes "github.com/cosmos/evm/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func init() {
	// manually update the power reduction by replacing micro (u) -> atto (a) evmos
	sdk.DefaultPowerReduction = cosmosevmtypes.AttoPowerReduction
}

// TESTD extends an ABCI application, but with most of its parameters exported.
type TESTD struct {
	*app.BASED
}

// NewExampleApp returns a reference to an initialized EVMD.
func NewTestApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	evmChainID uint64,
	evmAppOptions evmconfig.EVMOptionsFn,
	baseAppOptions ...func(*baseapp.BaseApp),
) *TESTD {
	evmd := &TESTD{
		BASED: app.NewApp(
			logger,
			db,
			traceStore,
			false,
			appOpts,
			evmChainID,
			evmAppOptions,
			baseAppOptions...,
		),
	}
	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))
	evmd.setAnteHandler(evmd.TxConfig(), maxGasWanted)

	if loadLatest {
		if err := evmd.LoadLatestVersion(); err != nil {
			logger.Error("error on loading last version", "err", err)
			os.Exit(1)
		}
	}
	return evmd
}

func (app *TESTD) setAnteHandler(txConfig client.TxConfig, maxGasWanted uint64) {
	options := HandlerOptions{
		Cdc:                    app.AppCodec(),
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		ExtensionOptionChecker: cosmosevmtypes.HasDynamicFeeExtensionOption,
		EvmKeeper:              app.EVMKeeper,
		FeegrantKeeper:         app.FeeGrantKeeper,
		IBCKeeper:              app.IBCKeeper,
		FeeMarketKeeper:        app.FeeMarketKeeper,
		SignModeHandler:        txConfig.SignModeHandler(),
		SigGasConsumer:         evmante.SigVerificationGasConsumer,
		MaxTxGasWanted:         maxGasWanted,
		TxFeeChecker:           cosmosevmante.NewDynamicFeeChecker(app.FeeMarketKeeper),
	}
	if err := options.Validate(); err != nil {
		panic(err)
	}

	app.SetAnteHandler(NewAnteHandler(options))
}
