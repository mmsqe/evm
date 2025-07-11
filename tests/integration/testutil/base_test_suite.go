package testutil

import (
	"github.com/stretchr/testify/suite"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/evm"
	"github.com/cosmos/evm/tests/integration/testutil/app"
	"github.com/cosmos/evm/testutil/config"
	"github.com/cosmos/evm/testutil/integration/evm/factory"
	"github.com/cosmos/evm/testutil/integration/evm/grpc"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/testutil/keyring"

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

type BaseTestSuiteWithFactory struct {
	BaseTestSuite

	Network *network.UnitTestNetwork
	Keyring keyring.Keyring
	Factory factory.TxFactory
}

func (s *BaseTestSuiteWithFactory) SetupTest() {
	s.BaseTestSuite.SetupTest()
	keys := keyring.New(2)
	opts := []network.ConfigOption{
		network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
	}
	s.Network = network.NewUnitTestNetwork(s.Create, opts...)
	gh := grpc.NewIntegrationHandler(s.Network)
	s.Factory = factory.New(s.Network, gh)
	s.Keyring = keys
}
