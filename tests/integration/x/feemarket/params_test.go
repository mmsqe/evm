package feemarket_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/evm/tests/integration/testutil"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/x/feemarket/types"
)

type ParamsTestSuite struct {
	testutil.BaseTestSuite
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (s *ParamsTestSuite) TestGetParams() {
	nw := network.NewUnitTestNetwork(s.Create)
	ctx := nw.GetContext()

	params := nw.App.GetFeeMarketKeeper().GetParams(ctx)
	s.NotNil(params.BaseFee)
	s.NotNil(params.MinGasPrice)
	s.NotNil(params.MinGasMultiplier)
}

func (s *ParamsTestSuite) TestSetGetParams() {
	nw := network.NewUnitTestNetwork(s.Create)
	ctx := nw.GetContext()

	params := types.DefaultParams()
	err := nw.App.GetFeeMarketKeeper().SetParams(ctx, params)
	s.NoError(err)

	testCases := []struct {
		name      string
		paramsFun func() interface{}
		getFun    func() interface{}
		expected  bool
	}{
		{
			"success - Checks if the default params are set correctly",
			func() interface{} {
				return types.DefaultParams()
			},
			func() interface{} {
				return nw.App.GetFeeMarketKeeper().GetParams(ctx)
			},
			true,
		},
		{
			"success - Check ElasticityMultiplier is set to 3 and can be retrieved correctly",
			func() interface{} {
				params.ElasticityMultiplier = 3
				err := nw.App.GetFeeMarketKeeper().SetParams(ctx, params)
				s.NoError(err)
				return params.ElasticityMultiplier
			},
			func() interface{} {
				return nw.App.GetFeeMarketKeeper().GetParams(ctx).ElasticityMultiplier
			},
			true,
		},
		{
			"success - Check BaseFeeEnabled is computed with its default params and can be retrieved correctly",
			func() interface{} {
				err := nw.App.GetFeeMarketKeeper().SetParams(ctx, types.DefaultParams())
				s.NoError(err)
				return true
			},
			func() interface{} {
				return nw.App.GetFeeMarketKeeper().GetBaseFeeEnabled(ctx)
			},
			true,
		},
		{
			"success - Check BaseFeeEnabled is computed with alternate params and can be retrieved correctly",
			func() interface{} {
				params.NoBaseFee = true
				params.EnableHeight = 5
				err := nw.App.GetFeeMarketKeeper().SetParams(ctx, params)
				s.NoError(err)
				return true
			},
			func() interface{} {
				return nw.App.GetFeeMarketKeeper().GetBaseFeeEnabled(ctx)
			},
			false,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			outcome := reflect.DeepEqual(tc.paramsFun(), tc.getFun())
			s.Equal(tc.expected, outcome)
		})
	}
}
