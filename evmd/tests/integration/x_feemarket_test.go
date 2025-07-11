package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/evm/tests/integration/testutil"
)

func TestFeeMarketKeeperTestSuite(t *testing.T) {
	suite.Run(t, &testutil.BaseTestSuite{Create: CreateEvmd})
}
