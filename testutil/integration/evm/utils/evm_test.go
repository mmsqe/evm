package utils_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/evm/contracts"
	"github.com/cosmos/evm/tests/integration/testutil"
	testfactory "github.com/cosmos/evm/testutil/integration/evm/factory"
	testhandler "github.com/cosmos/evm/testutil/integration/evm/grpc"
	testnetwork "github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/testutil/integration/evm/utils"
	testkeyring "github.com/cosmos/evm/testutil/keyring"
	testutiltypes "github.com/cosmos/evm/testutil/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

type UtilsEvmTestSuite struct {
	testutil.BaseTestSuite
}

func TestUtilsEvmTestSuite(t *testing.T) {
	suite.Run(t, new(UtilsEvmTestSuite))
}

func (s *UtilsEvmTestSuite) TestGetERC20Balance() {
	s.SetupTest()
	keyring := testkeyring.New(1)
	options := []testnetwork.ConfigOption{
		testnetwork.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	}
	network := testnetwork.NewUnitTestNetwork(s.Create, options...)
	handler := testhandler.NewIntegrationHandler(network)
	factory := testfactory.New(network, handler)

	sender := keyring.GetKey(0)
	mintAmount := big.NewInt(100)

	// Deploy an ERC-20 contract
	erc20Addr, err := factory.DeployContract(
		sender.Priv,
		evmtypes.EvmTxArgs{},
		testutiltypes.ContractDeploymentData{
			Contract:        contracts.ERC20MinterBurnerDecimalsContract,
			ConstructorArgs: []interface{}{"TestToken", "TT", uint8(18)},
		},
	)
	s.NoError(err, "failed to deploy contract")
	s.NoError(network.NextBlock(), "failed to advance block")

	balance, err := utils.GetERC20Balance(network, erc20Addr, sender.Addr)
	s.NoError(err, "failed to get ERC20 balance")
	s.Equal(common.Big0.Int64(), balance.Int64(), "expected no balance before minting")

	// Mint some tokens
	_, err = factory.ExecuteContractCall(
		sender.Priv,
		evmtypes.EvmTxArgs{
			To: &erc20Addr,
		},
		testutiltypes.CallArgs{
			ContractABI: contracts.ERC20MinterBurnerDecimalsContract.ABI,
			MethodName:  "mint",
			Args:        []interface{}{sender.Addr, mintAmount},
		},
	)
	s.NoError(err, "failed to mint tokens")

	s.NoError(network.NextBlock(), "failed to advance block")

	balance, err = utils.GetERC20Balance(network, erc20Addr, sender.Addr)
	s.NoError(err, "failed to get ERC20 balance")
	s.Equal(mintAmount.Int64(), balance.Int64(), "expected different balance after minting")
}
