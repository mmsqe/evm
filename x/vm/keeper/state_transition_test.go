package keeper_test

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/cometbft/cometbft/crypto/tmhash"
	cmttypes "github.com/cometbft/cometbft/types"

	exampleapp "github.com/cosmos/evm/evmd"
	"github.com/cosmos/evm/testutil/integration/os/factory"
	"github.com/cosmos/evm/testutil/integration/os/grpc"
	testkeyring "github.com/cosmos/evm/testutil/integration/os/keyring"
	"github.com/cosmos/evm/testutil/integration/os/network"
	"github.com/cosmos/evm/testutil/integration/os/utils"
	utiltx "github.com/cosmos/evm/testutil/tx"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	"github.com/cosmos/evm/x/vm/keeper"
	"github.com/cosmos/evm/x/vm/types"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (suite *KeeperTestSuite) TestContextSetConsensusParams() {
	// set new value of max gas in consensus params
	maxGas := int64(123456789)
	res, err := s.network.App.ConsensusParamsKeeper.Params(s.network.GetContext(), &consensustypes.QueryParamsRequest{})
	s.Require().NoError(err)
	consParams := res.Params
	consParams.Block.MaxGas = maxGas
	_, err = s.network.App.ConsensusParamsKeeper.UpdateParams(s.network.GetContext(), &consensustypes.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Block:     consParams.Block,
		Evidence:  consParams.Evidence,
		Validator: consParams.Validator,
		Abci:      consParams.Abci,
	})
	s.Require().NoError(err)

	queryContext := s.network.GetQueryContext()
	proposerAddress := queryContext.BlockHeader().ProposerAddress
	cfg, err := s.network.App.EVMKeeper.EVMConfig(queryContext, proposerAddress)
	s.Require().NoError(err)

	sender := suite.keyring.GetKey(0)
	recipient := suite.keyring.GetAddr(1)
	msg, err := suite.factory.GenerateGethCoreMsg(sender.Priv, types.EvmTxArgs{
		To:     &recipient,
		Amount: big.NewInt(100),
	})
	suite.Require().NoError(err)

	// evm should query the max gas from consensus keeper, yielding the number set above.
	vm := s.network.App.EVMKeeper.NewEVM(queryContext, *msg, cfg, nil, s.network.GetStateDB())
	//nolint:gosec
	s.Require().Equal(vm.Context.GasLimit, uint64(maxGas))

	// if we explicitly set the consensus params in context, like when Cosmos builds a transaction context,
	// we should use that value, and not query the consensus params from the keeper.
	consParams.Block.MaxGas = 54321
	queryContext = queryContext.WithConsensusParams(*consParams)
	vm = s.network.App.EVMKeeper.NewEVM(queryContext, *msg, cfg, nil, s.network.GetStateDB())
	//nolint:gosec
	s.Require().Equal(vm.Context.GasLimit, uint64(consParams.Block.MaxGas))
}

func (suite *KeeperTestSuite) TestGetHashFn() {
	suite.SetupTest()
	header := suite.network.GetContext().BlockHeader()
	cmtHeader, _ := cmttypes.HeaderFromProto(&header)
	hash := cmtHeader.Hash()

	testCases := []struct {
		msg      string
		height   uint64
		malleate func() sdk.Context
		expHash  common.Hash
	}{
		{
			"case 1: context hash cached",
			uint64(suite.network.GetContext().BlockHeight()), //nolint:gosec // G115
			func() sdk.Context {
				return suite.network.GetContext().WithHeaderHash(
					tmhash.Sum([]byte("header")),
				)
			},
			common.BytesToHash(tmhash.Sum([]byte("header"))),
		},
		{
			"case 2: height lower than current one, found in storage",
			1,
			func() sdk.Context {
				return suite.network.GetContext().WithBlockHeight(10)
			},
			common.BytesToHash(hash),
		},
		{
			"case 3: height greater than current one",
			200,
			func() sdk.Context { return suite.network.GetContext() },
			common.Hash{},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			ctx := tc.malleate()

			// Function being tested
			hash := suite.network.App.EVMKeeper.GetHashFn(ctx)(tc.height)
			suite.Require().Equal(tc.expHash, hash)

			err := suite.network.NextBlock()
			suite.Require().NoError(err)
		})
	}
}

func (suite *KeeperTestSuite) TestGetCoinbaseAddress() {
	suite.SetupTest()
	validators := suite.network.GetValidators()
	proposerAddressHex := utils.ValidatorConsAddressToHex(
		validators[0].OperatorAddress,
	)

	testCases := []struct {
		msg      string
		malleate func() sdk.Context
		expPass  bool
	}{
		{
			"validator not found",
			func() sdk.Context {
				header := suite.network.GetContext().BlockHeader()
				header.ProposerAddress = []byte{}
				return suite.network.GetContext().WithBlockHeader(header)
			},
			false,
		},
		{
			"success",
			func() sdk.Context {
				return suite.network.GetContext()
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			ctx := tc.malleate()
			proposerAddress := ctx.BlockHeader().ProposerAddress

			// Function being tested
			coinbase, err := suite.network.App.EVMKeeper.GetCoinbaseAddress(
				ctx,
				sdk.ConsAddress(proposerAddress),
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(proposerAddressHex, coinbase)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetEthIntrinsicGas() {
	suite.SetupTest()
	testCases := []struct {
		name               string
		data               []byte
		accessList         gethtypes.AccessList
		height             int64
		isContractCreation bool
		noError            bool
		expGas             uint64
	}{
		{
			"no data, no accesslist, not contract creation, not homestead, not istanbul, not shanghai",
			nil,
			nil,
			1,
			false,
			true,
			params.TxGas,
		},
		{
			"with one zero data, no accesslist, not contract creation, not homestead, not istanbul, not shanghai",
			[]byte{0},
			nil,
			1,
			false,
			true,
			params.TxGas + params.TxDataZeroGas*1,
		},
		{
			"with one non zero data, no accesslist, not contract creation, not homestead, not istanbul, not shanghai",
			[]byte{1},
			nil,
			1,
			true,
			true,
			params.TxGas + params.TxDataNonZeroGasFrontier*1,
		},
		{
			"no data, one accesslist, not contract creation, not homestead, not istanbul, not shanghai",
			nil,
			[]gethtypes.AccessTuple{
				{},
			},
			1,
			false,
			true,
			params.TxGas + params.TxAccessListAddressGas,
		},
		{
			"no data, one accesslist with one storageKey, not contract creation, not homestead, not istanbul, not shanghai",
			nil,
			[]gethtypes.AccessTuple{
				{StorageKeys: make([]common.Hash, 1)},
			},
			1,
			false,
			true,
			params.TxGas + params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas*1,
		},
		{
			"no data, no accesslist, is contract creation, is homestead, not istanbul, not shanghai",
			nil,
			nil,
			2,
			true,
			true,
			params.TxGasContractCreation,
		},
		{
			"with one zero data, no accesslist, not contract creation, is homestead, is istanbul, not shanghai",
			[]byte{1},
			nil,
			3,
			false,
			true,
			params.TxGas + params.TxDataNonZeroGasEIP2028*1,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			ethCfg := types.GetEthChainConfig()
			ethCfg.HomesteadBlock = big.NewInt(2)
			ethCfg.IstanbulBlock = big.NewInt(3)
			signer := gethtypes.LatestSignerForChainID(ethCfg.ChainID)

			// in the future, fork not enabled
			shanghaiTime := uint64(suite.network.GetContext().BlockTime().Unix()) + 10000 //#nosec G115 -- int overflow is not a concern here
			ethCfg.ShanghaiTime = &shanghaiTime

			ctx := suite.network.GetContext().WithBlockHeight(tc.height)

			addr := suite.keyring.GetAddr(0)
			krSigner := utiltx.NewSigner(suite.keyring.GetPrivKey(0))
			nonce := suite.network.App.EVMKeeper.GetNonce(ctx, addr)
			m, err := newNativeMessage(
				nonce,
				ctx.BlockHeight(),
				addr,
				ethCfg,
				krSigner,
				signer,
				gethtypes.AccessListTxType,
				tc.data,
				tc.accessList,
			)
			suite.Require().NoError(err)

			// Function being tested
			gas, err := suite.network.App.EVMKeeper.GetEthIntrinsicGas(
				ctx,
				*m,
				ethCfg,
				tc.isContractCreation,
			)

			if tc.noError {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			suite.Require().Equal(tc.expGas, gas)
		})
	}
}

func (suite *KeeperTestSuite) TestGasToRefund() {
	suite.SetupTest()
	testCases := []struct {
		name           string
		gasconsumed    uint64
		refundQuotient uint64
		expGasRefund   uint64
		expPanic       bool
	}{
		{
			"gas refund 5",
			5,
			1,
			5,
			false,
		},
		{
			"gas refund 10",
			10,
			1,
			10,
			false,
		},
		{
			"gas refund availableRefund",
			11,
			1,
			10,
			false,
		},
		{
			"gas refund quotient 0",
			11,
			0,
			0,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			vmdb := suite.network.GetStateDB()
			vmdb.AddRefund(10)

			if tc.expPanic {
				panicF := func() {
					//nolint:staticcheck
					keeper.GasToRefund(vmdb.GetRefund(), tc.gasconsumed, tc.refundQuotient)
				}
				suite.Require().Panics(panicF)
			} else {
				gr := keeper.GasToRefund(vmdb.GetRefund(), tc.gasconsumed, tc.refundQuotient)
				suite.Require().Equal(tc.expGasRefund, gr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRefundGas() {
	// FeeCollector account is pre-funded with enough tokens
	// for refund to work
	// NOTE: everything should happen within the same block for
	// feecollector account to remain funded
	baseDenom := types.GetEVMCoinDenom()

	coins := sdk.NewCoins(sdk.NewCoin(
		baseDenom,
		sdkmath.NewInt(6e18),
	))
	balances := []banktypes.Balance{
		{
			Address: authtypes.NewModuleAddress(authtypes.FeeCollectorName).String(),
			Coins:   coins,
		},
	}
	bankGenesis := banktypes.DefaultGenesisState()
	bankGenesis.Balances = balances
	customGenesis := network.CustomGenesisState{}
	customGenesis[banktypes.ModuleName] = bankGenesis

	keyring := testkeyring.New(2)
	unitNetwork := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
		network.WithCustomGenesis(customGenesis),
	)
	grpcHandler := grpc.NewIntegrationHandler(unitNetwork)
	txFactory := factory.New(unitNetwork, grpcHandler)

	sender := keyring.GetKey(0)
	recipient := keyring.GetAddr(1)

	testCases := []struct {
		name           string
		leftoverGas    uint64
		refundQuotient uint64
		noError        bool
		expGasRefund   uint64
		gasPrice       *big.Int
	}{
		{
			name:           "leftoverGas more than tx gas limit",
			leftoverGas:    params.TxGas + 1,
			refundQuotient: params.RefundQuotient,
			noError:        false,
			expGasRefund:   params.TxGas + 1,
		},
		{
			name:           "leftoverGas equal to tx gas limit, insufficient fee collector account",
			leftoverGas:    params.TxGas,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   0,
		},
		{
			name:           "leftoverGas less than to tx gas limit",
			leftoverGas:    params.TxGas - 1,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   0,
		},
		{
			name:           "no leftoverGas, refund half used gas ",
			leftoverGas:    0,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   params.TxGas / params.RefundQuotient,
		},
		{
			name:           "invalid GasPrice in message",
			leftoverGas:    0,
			refundQuotient: params.RefundQuotient,
			noError:        false,
			expGasRefund:   params.TxGas / params.RefundQuotient,
			gasPrice:       big.NewInt(-100),
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			coreMsg, err := txFactory.GenerateGethCoreMsg(
				sender.Priv,
				types.EvmTxArgs{
					To:       &recipient,
					Amount:   big.NewInt(100),
					GasPrice: tc.gasPrice,
				},
			)
			suite.Require().NoError(err)
			transactionGas := coreMsg.GasLimit

			vmdb := unitNetwork.GetStateDB()
			vmdb.AddRefund(params.TxGas)

			if tc.leftoverGas > transactionGas {
				return
			}

			gasUsed := transactionGas - tc.leftoverGas
			refund := keeper.GasToRefund(vmdb.GetRefund(), gasUsed, tc.refundQuotient)
			suite.Require().Equal(tc.expGasRefund, refund)

			err = unitNetwork.App.EVMKeeper.RefundGas(
				unitNetwork.GetContext(),
				*coreMsg,
				refund,
				unitNetwork.GetBaseDenom(),
			)

			if tc.noError {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestResetGasMeterAndConsumeGas() {
	suite.SetupTest()
	testCases := []struct {
		name        string
		gasConsumed uint64
		gasUsed     uint64
		expPanic    bool
	}{
		{
			"gas consumed 5, used 5",
			5,
			5,
			false,
		},
		{
			"gas consumed 5, used 10",
			5,
			10,
			false,
		},
		{
			"gas consumed 10, used 10",
			10,
			10,
			false,
		},
		{
			"gas consumed 11, used 10, NegativeGasConsumed panic",
			11,
			10,
			true,
		},
		{
			"gas consumed 1, used 10, overflow panic",
			1,
			math.MaxUint64,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			panicF := func() {
				gm := storetypes.NewGasMeter(10)
				gm.ConsumeGas(tc.gasConsumed, "")
				ctx := suite.network.GetContext().WithGasMeter(gm)
				suite.network.App.EVMKeeper.ResetGasMeterAndConsumeGas(ctx, tc.gasUsed)
			}

			if tc.expPanic {
				suite.Require().Panics(panicF)
			} else {
				suite.Require().NotPanics(panicF)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestEVMConfig() {
	suite.SetupTest()

	defaultChainEVMParams := exampleapp.NewEVMGenesisState().Params

	proposerAddress := suite.network.GetContext().BlockHeader().ProposerAddress
	cfg, err := suite.network.App.EVMKeeper.EVMConfig(
		suite.network.GetContext(),
		proposerAddress,
	)
	suite.Require().NoError(err)
	suite.Require().Equal(defaultChainEVMParams, cfg.Params)
	// london hardfork is enabled by default
	suite.Require().Equal(big.NewInt(0), cfg.BaseFee)

	validators := suite.network.GetValidators()
	proposerHextAddress := utils.ValidatorConsAddressToHex(validators[0].OperatorAddress)
	suite.Require().Equal(proposerHextAddress, cfg.CoinBase)
}

func (suite *KeeperTestSuite) TestApplyTransaction() {
	suite.enableFeemarket = true
	defer func() { suite.enableFeemarket = false }()
	// FeeCollector account is pre-funded with enough tokens
	// for refund to work
	// NOTE: everything should happen within the same block for
	// feecollector account to remain funded
	suite.SetupTest()
	// set bounded cosmos block gas limit
	ctx := suite.network.GetContext().WithBlockGasMeter(storetypes.NewGasMeter(1e6))
	err := suite.network.App.BankKeeper.MintCoins(ctx, "mint", sdk.NewCoins(sdk.NewCoin("aatom", sdkmath.NewInt(3e18))))
	suite.Require().NoError(err)
	err = suite.network.App.BankKeeper.SendCoinsFromModuleToModule(ctx, "mint", "fee_collector", sdk.NewCoins(sdk.NewCoin("aatom", sdkmath.NewInt(3e18))))
	suite.Require().NoError(err)
	testCases := []struct {
		name       string
		gasLimit   uint64
		requireErr bool
		errorMsg   string
	}{
		{
			"pass - set evm limit above cosmos block gas limit and refund",
			6e6,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			tx, err := suite.factory.GenerateSignedEthTx(suite.keyring.GetPrivKey(0), types.EvmTxArgs{
				GasLimit: tc.gasLimit,
			})
			suite.Require().NoError(err)
			initialBalance := suite.network.App.BankKeeper.GetBalance(ctx, suite.keyring.GetAccAddr(0), "aatom")

			ethTx := tx.GetMsgs()[0].(*types.MsgEthereumTx).AsTransaction()
			res, err := suite.network.App.EVMKeeper.ApplyTransaction(ctx, ethTx)
			suite.Require().NoError(err)
			suite.Require().Equal(res.GasUsed, uint64(3e6))
			// Half of the gas should be refunded based on the protocol refund cap.
			// Note that the balance should only increment by the refunded amount
			// because ApplyTransaction does not consume and take the gas from the user.
			balanceAfterRefund := suite.network.App.BankKeeper.GetBalance(ctx, suite.keyring.GetAccAddr(0), "aatom")
			expectedRefund := new(big.Int).Mul(new(big.Int).SetUint64(6e6/2), suite.network.App.EVMKeeper.GetBaseFee(ctx))
			suite.Require().Equal(balanceAfterRefund.Sub(initialBalance).Amount, sdkmath.NewIntFromBigInt(expectedRefund))
		})
	}
}

func (suite *KeeperTestSuite) TestApplyMessage() {
	suite.enableFeemarket = true
	defer func() { suite.enableFeemarket = false }()
	suite.SetupTest()

	// Generate a transfer tx message
	sender := suite.keyring.GetKey(0)
	recipient := suite.keyring.GetAddr(1)
	transferArgs := types.EvmTxArgs{
		To:     &recipient,
		Amount: big.NewInt(100),
	}
	coreMsg, err := suite.factory.GenerateGethCoreMsg(
		sender.Priv,
		transferArgs,
	)
	suite.Require().NoError(err)

	tracer := suite.network.App.EVMKeeper.Tracer(
		suite.network.GetContext(),
		*coreMsg,
		types.GetEthChainConfig(),
	)
	res, err := suite.network.App.EVMKeeper.ApplyMessage(suite.network.GetContext(), *coreMsg, tracer, true, false)
	suite.Require().NoError(err)
	suite.Require().False(res.Failed())

	// Compare gas to a transfer tx gas
	expectedGasUsed := params.TxGas
	suite.Require().Equal(expectedGasUsed, res.GasUsed)
}

func (suite *KeeperTestSuite) TestApplyMessageWithConfig() {
	suite.enableFeemarket = true
	defer func() { suite.enableFeemarket = false }()
	suite.SetupTest()
	testCases := []struct {
		name               string
		getMessage         func() core.Message
		getEVMParams       func() types.Params
		getFeeMarketParams func() feemarkettypes.Params
		expErr             bool
		expVMErr           bool
		expectedGasUsed    uint64
	}{
		{
			"success - messsage applied ok with default params",
			func() core.Message {
				sender := suite.keyring.GetKey(0)
				recipient := suite.keyring.GetAddr(1)
				msg, err := suite.factory.GenerateGethCoreMsg(sender.Priv, types.EvmTxArgs{
					To:     &recipient,
					Amount: big.NewInt(100),
				})
				suite.Require().NoError(err)
				return *msg
			},
			types.DefaultParams,
			feemarkettypes.DefaultParams,
			false,
			false,
			params.TxGas,
		},
		{
			"call contract tx with config param EnableCall = false",
			func() core.Message {
				sender := suite.keyring.GetKey(0)
				recipient := suite.keyring.GetAddr(1)
				msg, err := suite.factory.GenerateGethCoreMsg(sender.Priv, types.EvmTxArgs{
					To:     &recipient,
					Amount: big.NewInt(100),
					Input:  []byte("contract_data"),
				})
				suite.Require().NoError(err)
				return *msg
			},
			func() types.Params {
				defaultParams := types.DefaultParams()
				defaultParams.AccessControl = types.AccessControl{
					Call: types.AccessControlType{
						AccessType: types.AccessTypeRestricted,
					},
				}
				return defaultParams
			},
			feemarkettypes.DefaultParams,
			false,
			true,
			0,
		},
		{
			"create contract tx with config param EnableCreate = false",
			func() core.Message {
				sender := suite.keyring.GetKey(0)
				msg, err := suite.factory.GenerateGethCoreMsg(sender.Priv, types.EvmTxArgs{
					Amount: big.NewInt(100),
					Input:  []byte("contract_data"),
				})
				suite.Require().NoError(err)
				return *msg
			},
			func() types.Params {
				defaultParams := types.DefaultParams()
				defaultParams.AccessControl = types.AccessControl{
					Create: types.AccessControlType{
						AccessType: types.AccessTypeRestricted,
					},
				}
				return defaultParams
			},
			feemarkettypes.DefaultParams,
			false,
			true,
			0,
		},
		{
			"fail - fix panic when minimumGasUsed is not uint64",
			func() core.Message {
				sender := suite.keyring.GetKey(0)
				recipient := suite.keyring.GetAddr(1)
				msg, err := suite.factory.GenerateGethCoreMsg(sender.Priv, types.EvmTxArgs{
					To:     &recipient,
					Amount: big.NewInt(100),
				})
				suite.Require().NoError(err)
				return *msg
			},
			types.DefaultParams,
			func() feemarkettypes.Params {
				paramsRes, err := suite.handler.GetFeeMarketParams()
				suite.Require().NoError(err)
				params := paramsRes.GetParams()
				params.MinGasMultiplier = sdkmath.LegacyNewDec(math.MaxInt64).MulInt64(100)
				return params
			},
			true,
			false,
			0,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			msg := tc.getMessage()
			evmParams := tc.getEVMParams()
			err := suite.network.App.EVMKeeper.SetParams(
				suite.network.GetContext(),
				evmParams,
			)
			suite.Require().NoError(err)
			feeMarketparams := tc.getFeeMarketParams()
			err = suite.network.App.FeeMarketKeeper.SetParams(
				suite.network.GetContext(),
				feeMarketparams,
			)
			suite.Require().NoError(err)

			txConfig := suite.network.App.EVMKeeper.TxConfig(
				suite.network.GetContext(),
				common.Hash{},
			)
			proposerAddress := suite.network.GetContext().BlockHeader().ProposerAddress
			config, err := suite.network.App.EVMKeeper.EVMConfig(
				suite.network.GetContext(),
				proposerAddress,
			)
			suite.Require().NoError(err)

			// Function being tested
			res, err := suite.network.App.EVMKeeper.ApplyMessageWithConfig(suite.network.GetContext(), msg, nil, true, config, txConfig, false)

			if tc.expErr {
				suite.Require().Error(err)
			} else if !tc.expVMErr {
				suite.Require().NoError(err)
				suite.Require().False(res.Failed())
				suite.Require().Equal(tc.expectedGasUsed, res.GasUsed)
			}

			err = suite.network.NextBlock()
			if tc.expVMErr {
				suite.Require().NotEmpty(res.VmError)
				return
			}

			if tc.expVMErr {
				suite.Require().NotEmpty(res.VmError)
				return
			}

			suite.Require().NoError(err)
		})
	}
}

func (suite *KeeperTestSuite) TestGetProposerAddress() {
	suite.SetupTest()
	address := sdk.ConsAddress(suite.keyring.GetAddr(0).Bytes())
	proposerAddress := sdk.ConsAddress(suite.network.GetContext().BlockHeader().ProposerAddress)
	testCases := []struct {
		msg    string
		addr   sdk.ConsAddress
		expAdr sdk.ConsAddress
	}{
		{
			"proposer address provided",
			address,
			address,
		},
		{
			"nil proposer address provided",
			nil,
			proposerAddress,
		},
		{
			"typed nil proposer address provided",
			sdk.ConsAddress{},
			proposerAddress,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.Require().Equal(
				tc.expAdr,
				keeper.GetProposerAddress(suite.network.GetContext(), tc.addr),
			)
		})
	}
}

func (suite *KeeperTestSuite) TestApplyMessageWithNegativeAmount() {
	suite.enableFeemarket = true
	defer func() { suite.enableFeemarket = false }()
	suite.SetupTest()

	// Generate a transfer tx message
	sender := suite.keyring.GetKey(0)
	recipient := suite.keyring.GetAddr(1)
	amt, _ := big.NewInt(0).SetString("-115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)
	transferArgs := types.EvmTxArgs{
		To:     &recipient,
		Amount: amt,
	}
	coreMsg, err := suite.factory.GenerateGethCoreMsg(
		sender.Priv,
		transferArgs,
	)
	suite.Require().NoError(err)

	tracer := suite.network.App.EVMKeeper.Tracer(
		suite.network.GetContext(),
		*coreMsg,
		types.GetEthChainConfig(),
	)

	ctx := suite.network.GetContext()
	balance0Before := suite.network.App.BankKeeper.GetBalance(ctx, suite.keyring.GetAccAddr(0), "aatom")
	balance1Before := suite.network.App.BankKeeper.GetBalance(ctx, suite.keyring.GetAccAddr(1), "aatom")
	res, err := suite.network.App.EVMKeeper.ApplyMessage(
		suite.network.GetContext(),
		*coreMsg,
		tracer,
		true,
		false,
	)
	suite.Require().Nil(res)
	suite.Require().Error(err)

	balance0After := suite.network.App.BankKeeper.GetBalance(ctx, suite.keyring.GetAccAddr(0), "aatom")
	balance1After := suite.network.App.BankKeeper.GetBalance(ctx, suite.keyring.GetAccAddr(1), "aatom")

	suite.Require().Equal(balance0Before, balance0After)
	suite.Require().Equal(balance1Before, balance1After)
}
