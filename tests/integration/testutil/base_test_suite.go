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
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	"github.com/cosmos/evm/x/vm/statedb"

	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simutils "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/ethereum/go-ethereum/common"
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

type BaseTestSuiteWithNetwork struct {
	BaseTestSuite

	Network *network.UnitTestNetwork
	Keyring keyring.Keyring
}

func (s *BaseTestSuiteWithNetwork) setup(
	opts ...network.ConfigOption,
) {
	s.BaseTestSuite.SetupTest()
	s.Keyring = keyring.New(2)
	opts = append([]network.ConfigOption{
		network.WithPreFundedAccounts(s.Keyring.GetAllAccAddrs()...),
	}, opts...)
	s.Network = network.NewUnitTestNetwork(s.Create, opts...)
}

func (s *BaseTestSuiteWithNetwork) SetupTest() {
	s.setup()
}

type BaseTestSuiteWithNetworkAndFactory struct {
	BaseTestSuiteWithNetwork

	Factory factory.TxFactory
}

func (s *BaseTestSuiteWithNetworkAndFactory) SetupTest() {
	s.setup()
	gh := grpc.NewIntegrationHandler(s.Network)
	s.Factory = factory.New(s.Network, gh)
}

type BaseTestSuiteWithFactoryAndGenesis struct {
	BaseTestSuiteWithNetworkAndFactory
	EnableFeemarket bool
}

func (s *BaseTestSuiteWithFactoryAndGenesis) SetupTest() {
	// Set custom balance based on test params
	customGenesis := network.CustomGenesisState{}
	feemarketGenesis := feemarkettypes.DefaultGenesisState()
	if s.EnableFeemarket {
		feemarketGenesis.Params.EnableHeight = 1
		feemarketGenesis.Params.NoBaseFee = false
	} else {
		feemarketGenesis.Params.NoBaseFee = true
	}
	customGenesis[feemarkettypes.ModuleName] = feemarketGenesis
	s.setup(network.WithCustomGenesis(customGenesis))
}

func (s *BaseTestSuiteWithFactoryAndGenesis) StateDB() *statedb.StateDB {
	return statedb.New(s.Network.GetContext(), s.Network.App.GetEVMKeeper(), statedb.NewEmptyTxConfig(common.BytesToHash(s.Network.GetContext().HeaderHash())))
}
