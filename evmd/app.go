package evmd

import (
	"io"

	"cosmossdk.io/log"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/runtime"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmante "github.com/cosmos/evm/ante"
	cosmosevmante "github.com/cosmos/evm/ante/evm"
	"github.com/cosmos/evm/app"
	"github.com/cosmos/evm/evmd/ante"
	evmdconfig "github.com/cosmos/evm/evmd/cmd/evmd/config"
	srvflags "github.com/cosmos/evm/server/flags"
	cosmosevmtypes "github.com/cosmos/evm/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/spf13/cast"
)

func init() {
	// manually update the power reduction by replacing micro (u) -> atto (a) evmos
	sdk.DefaultPowerReduction = cosmosevmtypes.AttoPowerReduction

	defaultNodeHome = app.MustGetDefaultNodeHome()
}

// defaultNodeHome default home directories for the application daemon
var defaultNodeHome string

var (
	_ runtime.AppI            = (*EVMD)(nil)
	_ servertypes.Application = (*EVMD)(nil)
	_ ibctesting.TestingApp   = (*EVMD)(nil)
)

// EVMD extends an ABCI application, but with most of its parameters exported.
type EVMD struct {
	*app.BASED
}

// NewExampleApp returns a reference to an initialized EVMD.
func NewExampleApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	evmChainID uint64,
	evmAppOptions evmdconfig.EVMOptionsFn,
	baseAppOptions ...func(*baseapp.BaseApp),
) *EVMD {
	evmd := &EVMD{
		BASED: app.NewApp(
			logger,
			db,
			traceStore,
			loadLatest,
			appOpts,
			evmChainID,
			evmAppOptions,
			baseAppOptions...,
		),
	}
	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))
	evmd.setAnteHandler(evmd.TxConfig(), maxGasWanted)
	return evmd
}

func (app *EVMD) setAnteHandler(txConfig client.TxConfig, maxGasWanted uint64) {
	options := ante.HandlerOptions{
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

	app.SetAnteHandler(ante.NewAnteHandler(options))
}
