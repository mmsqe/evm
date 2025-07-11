package feemarket_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/evm/tests/integration/testutil"
)

type KeeperTestSuite struct {
	testutil.BaseTestSuite
}

func TestFeeMarketKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
