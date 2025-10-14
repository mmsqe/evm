//go:build !ledger_zemu
// +build !ledger_zemu

package keys

import (
	cosmosLedger "github.com/cosmos/cosmos-sdk/crypto/ledger"
	evmkeyring "github.com/cosmos/evm/crypto/keyring"
)

func getEthereumLedgerDiscovery() func() (cosmosLedger.SECP256K1, error) {
	return func() (cosmosLedger.SECP256K1, error) {
		return evmkeyring.LedgerDerivation()
	}
}
