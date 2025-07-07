package testutil

import (
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/testutil/integration/evm/network"
)

type BaseTestSuite struct {
	suite.Suite

	Create network.CreateEvmApp
}

func (suite *BaseTestSuite) SetupTest() {
	suite.Create = integration.CreateEvmd
}
