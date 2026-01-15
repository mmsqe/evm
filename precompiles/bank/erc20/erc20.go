package erc20

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	cmn "github.com/cosmos/evm/precompiles/common"
)

var ERC20ABI = []string{
	"function name() view returns (string name)",
	"function symbol() view returns (string symbol)",
	"function decimals() view returns (uint8 decimals)",
	"function totalSupply() view returns (uint256 supply)",
	"function balanceOf(address account) view returns (uint256 balance)",
	"function transfer(address to, uint256 amount) returns (bool success)",
	"function transferFrom(address from, address to, uint256 amount) returns (bool success)",
	"event Transfer(address indexed from, address indexed to, uint256 amount)",
	"event Approval(address indexed owner, address indexed spender, uint256 amount)",
}

var parsedABI abi.ABI

func init() {
	var err error
	parsedABI, err = cmn.ParseHumanReadableABI(ERC20ABI)
	if err != nil {
		panic(err)
	}
}

// Call helper functions
func NewNameCall() []byte {
	data, _ := parsedABI.Pack("name")
	return data
}

func NewSymbolCall() []byte {
	data, _ := parsedABI.Pack("symbol")
	return data
}

func NewDecimalsCall() []byte {
	data, _ := parsedABI.Pack("decimals")
	return data
}

func NewTotalSupplyCall() []byte {
	data, _ := parsedABI.Pack("totalSupply")
	return data
}

func NewBalanceOfCall(account common.Address) []byte {
	data, _ := parsedABI.Pack("balanceOf", account)
	return data
}

func NewTransferCall(to common.Address, amount *big.Int) []byte {
	data, _ := parsedABI.Pack("transfer", to, amount)
	return data
}

func NewTransferFromCall(from, to common.Address, amount *big.Int) []byte {
	data, _ := parsedABI.Pack("transferFrom", from, to, amount)
	return data
}

// Return types for decoding
type NameReturn struct {
	Name string
}

type SymbolReturn struct {
	Symbol string
}

type DecimalsReturn struct {
	Decimals uint8
}

type TotalSupplyReturn struct {
	Supply *big.Int
}

type BalanceOfReturn struct {
	Balance *big.Int
}

type TransferReturn struct {
	Success bool
}

type TransferFromReturn struct {
	Success bool
}

// Decode helpers
func DecodeNameReturn(data []byte) (*NameReturn, error) {
	var result NameReturn
	err := parsedABI.UnpackIntoInterface(&result, "name", data)
	return &result, err
}

func DecodeSymbolReturn(data []byte) (*SymbolReturn, error) {
	var result SymbolReturn
	err := parsedABI.UnpackIntoInterface(&result, "symbol", data)
	return &result, err
}

func DecodeDecimalsReturn(data []byte) (*DecimalsReturn, error) {
	var result DecimalsReturn
	err := parsedABI.UnpackIntoInterface(&result, "decimals", data)
	return &result, err
}

func DecodeTotalSupplyReturn(data []byte) (*TotalSupplyReturn, error) {
	var result TotalSupplyReturn
	err := parsedABI.UnpackIntoInterface(&result, "totalSupply", data)
	return &result, err
}

func DecodeBalanceOfReturn(data []byte) (*BalanceOfReturn, error) {
	var result BalanceOfReturn
	err := parsedABI.UnpackIntoInterface(&result, "balanceOf", data)
	return &result, err
}

func DecodeTransferReturn(data []byte) (*TransferReturn, error) {
	var result TransferReturn
	err := parsedABI.UnpackIntoInterface(&result, "transfer", data)
	return &result, err
}

func DecodeTransferFromReturn(data []byte) (*TransferFromReturn, error) {
	var result TransferFromReturn
	err := parsedABI.UnpackIntoInterface(&result, "transferFrom", data)
	return &result, err
}

// Encode methods for test compatibility
func (r *NameReturn) Encode() ([]byte, error) {
	return parsedABI.Methods["name"].Outputs.Pack(r.Name)
}

func (r *SymbolReturn) Encode() ([]byte, error) {
	return parsedABI.Methods["symbol"].Outputs.Pack(r.Symbol)
}

func (r *DecimalsReturn) Encode() ([]byte, error) {
	return parsedABI.Methods["decimals"].Outputs.Pack(r.Decimals)
}

func (r *TotalSupplyReturn) Encode() ([]byte, error) {
	return parsedABI.Methods["totalSupply"].Outputs.Pack(r.Supply)
}

func (r *BalanceOfReturn) Encode() ([]byte, error) {
	return parsedABI.Methods["balanceOf"].Outputs.Pack(r.Balance)
}

func (r *TransferReturn) Encode() ([]byte, error) {
	return parsedABI.Methods["transfer"].Outputs.Pack(r.Success)
}

func (r *TransferFromReturn) Encode() ([]byte, error) {
	return parsedABI.Methods["transferFrom"].Outputs.Pack(r.Success)
}
