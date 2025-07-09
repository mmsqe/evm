package testutil

import (
	"github.com/stretchr/testify/suite"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/evm"
	"github.com/cosmos/evm/tests/integration/testutil/app"
	"github.com/cosmos/evm/testutil/config"
	"github.com/cosmos/evm/testutil/integration/evm/network"

	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simutils "github.com/cosmos/cosmos-sdk/testutil/sims"
)

type BaseTestSuite struct {
	suite.Suite

	Create network.CreateEvmApp
}

func (suite *BaseTestSuite) SetupTest() {
	suite.Create = func(chainID string, evmChainID uint64, customBaseAppOptions ...func(*baseapp.BaseApp)) evm.EvmApp {
		defaultNodeHome, err := clienthelpers.GetNodeHomeDirectory(".evmd")
		if err != nil {
			panic(err)
		}
		db := dbm.NewMemDB()
		logger := log.NewNopLogger()
		loadLatest := true
		appOptions := simutils.NewAppOptionsWithFlagHome(defaultNodeHome)
		baseAppOptions := append(customBaseAppOptions, baseapp.SetChainID(chainID)) //nolint:gocritic

		return app.NewTestApp(
			logger,
			db,
			nil,
			loadLatest,
			appOptions,
			evmChainID,
			config.EvmAppOptions,
			baseAppOptions...,
		)
	}
}
