package types

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName string name of module
	ModuleName = "evm"

	// StoreKey key for ethereum storage data, account code (StateDB) or block
	// related data for Web3.
	// The EVM module should use a prefix store.
	StoreKey = ModuleName

	// TransientKey is the key to access the EVM transient store, that is reset
	// during the Commit phase.
	TransientKey = "transient_" + ModuleName

	// RouterKey uses module name for routing
	RouterKey = ModuleName
)

// prefix bytes for the EVM persistent store
const (
	prefixCode = iota + 1
	prefixStorage
	prefixParams
	prefixCodeHash
	prefixHeaderHash
)

// prefix bytes for the EVM transient store
const (
	prefixTransientBloom = iota + 1
	prefixTransientTxIndex
	prefixTransientLogSize
	prefixTransientGasUsed
)

// KVStore key prefixes
var (
	KeyPrefixCode       = []byte{prefixCode}
	KeyPrefixStorage    = []byte{prefixStorage}
	KeyPrefixParams     = []byte{prefixParams}
	KeyPrefixCodeHash   = []byte{prefixCodeHash}
	KeyPrefixHeaderHash = []byte{prefixHeaderHash}
)

// Transient Store key prefixes
var (
	KeyPrefixTransientBloom   = []byte{prefixTransientBloom}
	KeyPrefixTransientTxIndex = []byte{prefixTransientTxIndex}
	KeyPrefixTransientLogSize = []byte{prefixTransientLogSize}
	KeyPrefixTransientGasUsed = []byte{prefixTransientGasUsed}
)

// AddressStoragePrefix returns a prefix to iterate over a given account storage.
func AddressStoragePrefix(address common.Address) []byte {
	return append(KeyPrefixStorage, address.Bytes()...)
}

// StateKey defines the full key under which an account state is stored.
func StateKey(address common.Address, key []byte) []byte {
	return append(AddressStoragePrefix(address), key...)
}

func GetHeaderHashKey(height int64) []byte {
	var key [1 + 8]byte
	key[0] = prefixHeaderHash
	binary.BigEndian.PutUint64(key[1:], uint64(height)) //nolint:gosec // G115 // block number won't exceed uint64
	return key[:]
}
