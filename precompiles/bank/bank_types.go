package bank

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// mustParseType is a helper to parse ABI types, panics on error
func mustParseType(typ string) abi.Type {
	t, err := abi.NewType(typ, "", nil)
	if err != nil {
		panic(fmt.Sprintf("failed to parse type %s: %v", typ, err))
	}
	return t
}

// Balance represents a token balance with contract address and amount
type Balance struct {
	ContractAddress common.Address
	Amount          *big.Int
}

// Call types (input parameters)
type BalancesCall struct {
	Account common.Address
}

type TotalSupplyCall struct{}

type SupplyOfCall struct {
	Contract common.Address
}

type NameCall struct {
	Denom string
}

type SymbolCall struct {
	Denom string
}

type DecimalsCall struct {
	Denom string
}

type TotalSupply0Call struct {
	Denom string
}

type BalanceOfCall struct {
	Account common.Address
	Denom   string
}

type TransferFromCall struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Denom string
}

type Erc20ctorCall struct {
	Denom string
	Bank  common.Address
}

// Return types (output parameters)
type BalancesReturn struct {
	Balances []Balance
}

type TotalSupplyReturn struct {
	TotalSupply []Balance
}

type SupplyOfReturn struct {
	TotalSupply *big.Int
}

type NameReturn struct {
	Name string
}

type SymbolReturn struct {
	Symbol string
}

type DecimalsReturn struct {
	Decimals uint8
}

type TotalSupply0Return struct {
	Supply *big.Int
}

type BalanceOfReturn struct {
	Balance *big.Int
}

type TransferFromReturn struct {
	Success bool
}

// Method ID constants for backward compatibility
var (
	BalancesID     = methodIDFromSig("balances(address)")
	TotalSupplyID  = methodIDFromSig("totalSupply()")
	SupplyOfID     = methodIDFromSig("supplyOf(address)")
	NameID         = methodIDFromSig("name(string)")
	SymbolID       = methodIDFromSig("symbol(string)")
	DecimalsID     = methodIDFromSig("decimals(string)")
	TotalSupply0ID = methodIDFromSig("totalSupply(string)")
	BalanceOfID    = methodIDFromSig("balanceOf(address,string)")
	TransferFromID = methodIDFromSig("transferFrom(address,address,uint256,string)")
	Erc20ctorID    = methodIDFromSig("erc20ctor(string,address)")
)

func methodIDFromSig(sig string) uint32 {
	// Compute method ID directly from signature using Keccak256
	// This is the standard way Ethereum computes method IDs: first 4 bytes of Keccak256 hash
	hash := crypto.Keccak256([]byte(sig))
	return uint32(hash[0])<<24 | uint32(hash[1])<<16 | uint32(hash[2])<<8 | uint32(hash[3])
}

// Decode implementations
//
// Note: TotalSupply0Call uses manual decoding because of method overloading.
// The bank precompile has two totalSupply methods:
//   - totalSupply() returns Balance[]
//   - totalSupply(string) returns uint256
//
// Since ABI.Methods is keyed by name, we manually decode totalSupply(string)
// to avoid conflicts.
func (c *BalancesCall) Decode(data []byte) (int, error) {
	unpacked, err := ABI.Methods["balances"].Inputs.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) == 0 {
		return 0, fmt.Errorf("no arguments provided")
	}
	c.Account = unpacked[0].(common.Address)
	return len(data), nil
}

func (c *TotalSupplyCall) Decode(data []byte) (int, error) {
	return len(data), nil
}

func (c *SupplyOfCall) Decode(data []byte) (int, error) {
	unpacked, err := ABI.Methods["supplyOf"].Inputs.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) == 0 {
		return 0, fmt.Errorf("no arguments provided")
	}
	c.Contract = unpacked[0].(common.Address)
	return len(data), nil
}

func (c *NameCall) Decode(data []byte) (int, error) {
	unpacked, err := ABI.Methods["name"].Inputs.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) == 0 {
		return 0, fmt.Errorf("no arguments provided")
	}
	c.Denom = unpacked[0].(string)
	return len(data), nil
}

func (c *SymbolCall) Decode(data []byte) (int, error) {
	unpacked, err := ABI.Methods["symbol"].Inputs.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) == 0 {
		return 0, fmt.Errorf("no arguments provided")
	}
	c.Denom = unpacked[0].(string)
	return len(data), nil
}

func (c *DecimalsCall) Decode(data []byte) (int, error) {
	unpacked, err := ABI.Methods["decimals"].Inputs.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) == 0 {
		return 0, fmt.Errorf("no arguments provided")
	}
	c.Denom = unpacked[0].(string)
	return len(data), nil
}

func (c *TotalSupply0Call) Decode(data []byte) (int, error) {
	// Manually decode since we have method overloading for totalSupply
	// totalSupply(string) has 1 string parameter
	unpacked, err := abi.Arguments{{Type: mustParseType("string")}}.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) == 0 {
		return 0, fmt.Errorf("no arguments provided")
	}
	c.Denom = unpacked[0].(string)
	return len(data), nil
}

func (c *BalanceOfCall) Decode(data []byte) (int, error) {
	unpacked, err := ABI.Methods["balanceOf"].Inputs.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) < 2 {
		return 0, fmt.Errorf("insufficient arguments provided")
	}
	c.Account = unpacked[0].(common.Address)
	c.Denom = unpacked[1].(string)
	return len(data), nil
}

func (c *TransferFromCall) Decode(data []byte) (int, error) {
	unpacked, err := ABI.Methods["transferFrom"].Inputs.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) < 4 {
		return 0, fmt.Errorf("insufficient arguments provided")
	}
	c.From = unpacked[0].(common.Address)
	c.To = unpacked[1].(common.Address)
	c.Value = unpacked[2].(*big.Int)
	c.Denom = unpacked[3].(string)
	return len(data), nil
}

func (c *Erc20ctorCall) Decode(data []byte) (int, error) {
	unpacked, err := ABI.Methods["erc20ctor"].Inputs.Unpack(data)
	if err != nil {
		return 0, err
	}
	if len(unpacked) < 2 {
		return 0, fmt.Errorf("insufficient arguments provided")
	}
	c.Denom = unpacked[0].(string)
	c.Bank = unpacked[1].(common.Address)
	return len(data), nil
}

// Encode implementations
//
// Note: TotalSupply0Return uses manual encoding because of method overloading.
// See the Decode section comment for details.
func (r *BalancesReturn) Encode() ([]byte, error) {
	// Convert []Balance to interface{} slice for ABI packing
	balances := make([]struct {
		ContractAddress common.Address
		Amount          *big.Int
	}, len(r.Balances))
	for i, b := range r.Balances {
		balances[i].ContractAddress = b.ContractAddress
		balances[i].Amount = b.Amount
	}
	return ABI.Methods["balances"].Outputs.Pack(balances)
}

func (r *TotalSupplyReturn) Encode() ([]byte, error) {
	totalSupply := make([]struct {
		ContractAddress common.Address
		Amount          *big.Int
	}, len(r.TotalSupply))
	for i, b := range r.TotalSupply {
		totalSupply[i].ContractAddress = b.ContractAddress
		totalSupply[i].Amount = b.Amount
	}
	return ABI.Methods["totalSupply"].Outputs.Pack(totalSupply)
}

func (r *SupplyOfReturn) Encode() ([]byte, error) {
	return ABI.Methods["supplyOf"].Outputs.Pack(r.TotalSupply)
}

func (r *NameReturn) Encode() ([]byte, error) {
	return ABI.Methods["name"].Outputs.Pack(r.Name)
}

func (r *SymbolReturn) Encode() ([]byte, error) {
	return ABI.Methods["symbol"].Outputs.Pack(r.Symbol)
}

func (r *DecimalsReturn) Encode() ([]byte, error) {
	return ABI.Methods["decimals"].Outputs.Pack(r.Decimals)
}

func (r *TotalSupply0Return) Encode() ([]byte, error) {
	return abi.Arguments{{Type: mustParseType("uint256")}}.Pack(r.Supply)
}

func (r *BalanceOfReturn) Encode() ([]byte, error) {
	return ABI.Methods["balanceOf"].Outputs.Pack(r.Balance)
}

func (r *TransferFromReturn) Encode() ([]byte, error) {
	return ABI.Methods["transferFrom"].Outputs.Pack(r.Success)
}

// EncodeTo implementations
//
// These methods implement the full Encode interface by delegating to Encode()
// and copying the result to the provided buffer.

func (r *BalancesReturn) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *TotalSupplyReturn) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *SupplyOfReturn) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *NameReturn) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *SymbolReturn) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *DecimalsReturn) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *TotalSupply0Return) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *BalanceOfReturn) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *TransferFromReturn) EncodeTo(buf []byte) (int, error) {
	data, err := r.Encode()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (r *BalancesReturn) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

func (r *TotalSupplyReturn) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

func (r *SupplyOfReturn) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

func (r *NameReturn) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

func (r *SymbolReturn) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

func (r *DecimalsReturn) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

func (r *TotalSupply0Return) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

func (r *BalanceOfReturn) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

func (r *TransferFromReturn) EncodedSize() int {
	data, _ := r.Encode()
	return len(data)
}

// Helper function for erc20ctor - returns a struct with Encode method
type Erc20ctorCallData struct {
	data []byte
}

func (e Erc20ctorCallData) Encode() ([]byte, error) {
	return e.data, nil
}

func NewErc20ctorCall(denom string, bank common.Address) Erc20ctorCallData {
	// For constructor, we only need the ABI-encoded arguments, not the method selector
	// Use the inputs from the erc20ctor "function" to encode just the parameters
	args := ABI.Methods["erc20ctor"].Inputs
	data, _ := args.Pack(denom, bank)
	return Erc20ctorCallData{data: data}
}

// Helper functions for creating ABI-encoded calls

func NewBalancesCall(account common.Address) []byte {
	data, _ := ABI.Pack("balances", account)
	return data
}

func NewTotalSupplyCall() []byte {
	data, _ := ABI.Pack("totalSupply")
	return data
}

func NewSupplyOfCall(contract common.Address) []byte {
	data, _ := ABI.Pack("supplyOf", contract)
	return data
}

func NewNameCall(denom string) []byte {
	data, _ := ABI.Pack("name", denom)
	return data
}

func NewSymbolCall(denom string) []byte {
	data, _ := ABI.Pack("symbol", denom)
	return data
}

func NewDecimalsCall(denom string) []byte {
	data, _ := ABI.Pack("decimals", denom)
	return data
}

func NewTotalSupply0Call(denom string) []byte {
	data, _ := ABI.Pack("totalSupply", denom)
	return data
}

func NewBalanceOfCall(account common.Address, denom string) []byte {
	data, _ := ABI.Pack("balanceOf", account, denom)
	return data
}

func NewTransferFromCall(from, to common.Address, value *big.Int, denom string) []byte {
	data, _ := ABI.Pack("transferFrom", from, to, value, denom)
	return data
}
