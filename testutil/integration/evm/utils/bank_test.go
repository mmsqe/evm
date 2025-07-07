package utils_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/evm/tests/integration/testutil"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/testutil/integration/evm/utils"
	testkeyring "github.com/cosmos/evm/testutil/keyring"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type UtilsBankTestSuite struct {
	testutil.BaseTestSuite
}

func TestUtilsBankTestSuite(t *testing.T) {
	suite.Run(t, new(UtilsBankTestSuite))
}

func (s *UtilsBankTestSuite) TestCheckBalances() {
	s.SetupTest()
	testDenom := "atest"
	keyring := testkeyring.New(1)
	address := keyring.GetAccAddr(0).String()

	testcases := []struct {
		name        string
		decimals    uint8
		expAmount   math.Int
		expPass     bool
		errContains string
	}{
		{
			name:      "pass - eighteen decimals",
			decimals:  18,
			expAmount: network.GetInitialAmount(evmtypes.EighteenDecimals),
			expPass:   true,
		},
		{
			name:      "pass - six decimals",
			decimals:  6,
			expAmount: network.GetInitialAmount(evmtypes.SixDecimals),
			expPass:   true,
		},
		{
			name:        "fail - wrong amount",
			decimals:    18,
			expAmount:   math.NewInt(1),
			errContains: "expected balance",
		},
	}

	for _, tc := range testcases {
		balances := []banktypes.Balance{{
			Address: address,
			Coins: sdk.NewCoins(
				sdk.NewCoin(testDenom, tc.expAmount),
			),
		}}

		options := []network.ConfigOption{
			network.WithBaseCoin(testDenom, tc.decimals),
			network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
		}
		nw := network.New(s.Create, options...)
		err := utils.CheckBalances(nw.GetContext(), nw.GetBankClient(), balances)
		if tc.expPass {
			s.NoError(err, "unexpected error checking balances")
		} else {
			s.Error(err, "expected error checking balances")
			s.ErrorContains(err, tc.errContains, "expected different error checking balances")
		}
	}
}
