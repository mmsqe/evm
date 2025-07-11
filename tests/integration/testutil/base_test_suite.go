package testutil

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/evm"
	servercfg "github.com/cosmos/evm/server/config"
	"github.com/cosmos/evm/tests/integration/testutil/app"
	"github.com/cosmos/evm/testutil/config"
	"github.com/cosmos/evm/testutil/integration/evm/factory"
	"github.com/cosmos/evm/testutil/integration/evm/grpc"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/testutil/keyring"
	utiltx "github.com/cosmos/evm/testutil/tx"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	"github.com/cosmos/evm/x/vm/keeper/testdata"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	clienthelpers "cosmossdk.io/client/v2/helpers"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simutils "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

	Handler grpc.Handler
	Factory factory.TxFactory
}

func (s *BaseTestSuiteWithNetworkAndFactory) setup(
	opts ...network.ConfigOption,
) {
	s.BaseTestSuiteWithNetwork.setup(opts...)
	s.Handler = grpc.NewIntegrationHandler(s.Network)
	s.Factory = factory.New(s.Network, s.Handler)
}

func (s *BaseTestSuiteWithNetworkAndFactory) SetupTest() {
	s.setup()
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

// DeployTestContract deploy a test erc20 contract and returns the contract address
func (s *BaseTestSuiteWithFactoryAndGenesis) DeployTestContract(t require.TestingT, ctx sdk.Context, owner common.Address, supply *big.Int) common.Address {
	chainID := evmtypes.GetEthChainConfig().ChainID

	erc20Contract, err := testdata.LoadERC20Contract()
	require.NoError(t, err, "failed to load contract")

	ctorArgs, err := erc20Contract.ABI.Pack("", owner, supply)
	require.NoError(t, err)

	addr := s.Keyring.GetAddr(0)
	nonce := s.Network.App.GetEVMKeeper().GetNonce(s.Network.GetContext(), addr)

	data := erc20Contract.Bin
	data = append(data, ctorArgs...)
	args, err := json.Marshal(&evmtypes.TransactionArgs{
		From: &addr,
		Data: (*hexutil.Bytes)(&data),
	})
	require.NoError(t, err)
	res, err := s.Network.GetEvmClient().EstimateGas(ctx, &evmtypes.EthCallRequest{
		Args:            args,
		GasCap:          servercfg.DefaultGasCap,
		ProposerAddress: s.Network.GetContext().BlockHeader().ProposerAddress,
	})
	require.NoError(t, err)

	baseFeeRes, err := s.Network.GetEvmClient().BaseFee(ctx, &evmtypes.QueryBaseFeeRequest{})
	require.NoError(t, err)

	var erc20DeployTx *evmtypes.MsgEthereumTx
	if s.EnableFeemarket {
		ethTxParams := &evmtypes.EvmTxArgs{
			ChainID:   chainID,
			Nonce:     nonce,
			GasLimit:  res.Gas,
			GasFeeCap: baseFeeRes.BaseFee.BigInt(),
			GasTipCap: big.NewInt(1),
			Input:     data,
			Accesses:  &ethtypes.AccessList{},
		}
		erc20DeployTx = evmtypes.NewTx(ethTxParams)
	} else {
		ethTxParams := &evmtypes.EvmTxArgs{
			ChainID:  chainID,
			Nonce:    nonce,
			GasLimit: res.Gas,
			Input:    data,
		}
		erc20DeployTx = evmtypes.NewTx(ethTxParams)
	}

	krSigner := utiltx.NewSigner(s.Keyring.GetPrivKey(0))
	erc20DeployTx.From = addr.Hex()
	err = erc20DeployTx.Sign(ethtypes.LatestSignerForChainID(chainID), krSigner)
	require.NoError(t, err)
	rsp, err := s.Network.App.GetEVMKeeper().EthereumTx(ctx, erc20DeployTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return crypto.CreateAddress(addr, nonce)
}
