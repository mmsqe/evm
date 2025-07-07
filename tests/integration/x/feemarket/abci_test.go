package feemarket_test

import (
	"testing"

	"github.com/cosmos/evm"
	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/testutil/integration/evm/network"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testconstants "github.com/cosmos/evm/testutil/constants"
	"github.com/stretchr/testify/suite"
)

type ABCITestSuite struct {
	suite.Suite

	create network.CreateEvmApp
}

func TestABCITestSuite(t *testing.T) {
	suite.Run(t, new(ABCITestSuite))
}

func (suite *ABCITestSuite) SetupTest() {
	suite.create = func(chainID string, evmChainID uint64, customBaseAppOptions ...func(*baseapp.BaseApp)) evm.EvmApp {
		return integration.CreateEvmd(testconstants.ExampleChainID.ChainID, testconstants.ExampleChainID.EVMChainID)
	}
}

func (s *ABCITestSuite) TestEndBlock() {
	s.SetupTest()
	var (
		nw  *network.UnitTestNetwork
		ctx sdk.Context
	)

	testCases := []struct {
		name         string
		NoBaseFee    bool
		malleate     func()
		expGasWanted uint64
	}{
		{
			"baseFee nil",
			true,
			func() {},
			uint64(0),
		},
		{
			"pass",
			false,
			func() {
				meter := storetypes.NewGasMeter(uint64(1000000000))
				ctx = ctx.WithBlockGasMeter(meter)
				nw.App.GetFeeMarketKeeper().SetTransientBlockGasWanted(ctx, 5000000)
			},
			uint64(2500000),
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// reset network and context
			nw = network.NewUnitTestNetwork(s.create)
			ctx = nw.GetContext()

			params := nw.App.GetFeeMarketKeeper().GetParams(ctx)
			params.NoBaseFee = tc.NoBaseFee

			err := nw.App.GetFeeMarketKeeper().SetParams(ctx, params)
			s.NoError(err)

			tc.malleate()

			err = nw.App.GetFeeMarketKeeper().EndBlock(ctx)
			s.NoError(err)

			gasWanted := nw.App.GetFeeMarketKeeper().GetBlockGasWanted(ctx)
			s.Equal(tc.expGasWanted, gasWanted, tc.name)
		})
	}
}
