//go:build ledger_zemu
// +build ledger_zemu

package keys

import (
	"encoding/hex"
	"fmt"

	cosmosLedger "github.com/cosmos/cosmos-sdk/crypto/ledger"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	ledger "github.com/cosmos/ledger-cosmos-go"
	ledger_go "github.com/zondax/ledger-go"
)

func getEthereumLedgerDiscovery() func() (cosmosLedger.SECP256K1, error) {
	return func() (cosmosLedger.SECP256K1, error) {
		zemuWrapper, err := findZemuCosmosAppSkipVersion()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Speculos Cosmos app: %w", err)
		}
		return &ZemuEVMWrapper{cosmosDevice: zemuWrapper}, nil
	}
}

func compressPublicKey(pubkey []byte) []byte {
	if len(pubkey) == 65 && pubkey[0] == 0x04 {
		compressed := make([]byte, 33)
		compressed[0] = 0x02 + (pubkey[64] & 1) // 0x02 if y is even, 0x03 if y is odd
		copy(compressed[1:], pubkey[1:33])      // x coordinate
		return compressed
	}
	return pubkey
}

func serializeHDPath(hdPath []uint32) []byte {
	if len(hdPath) >= 3 {
		hdPath[0] = hdPath[0] | 0x80000000 // Make coin_type hardened
		hdPath[1] = hdPath[1] | 0x80000000 // Make coin_type hardened
		hdPath[2] = hdPath[2] | 0x80000000 // Make account hardened
	}
	result := make([]byte, 1+4*len(hdPath))
	result[0] = byte(len(hdPath))
	for i, segment := range hdPath {
		offset := 1 + 4*i
		result[offset] = byte(segment >> 24)
		result[offset+1] = byte(segment >> 16)
		result[offset+2] = byte(segment >> 8)
		result[offset+3] = byte(segment)
	}
	return result
}

// ZemuLedgerCosmosWrapper implements all ledger.LedgerCosmos methods but skips version check
type ZemuLedgerCosmosWrapper struct {
	api     ledger_go.LedgerDevice
	version ledger.VersionInfo
}

// GetVersion returns a fake version without calling the device (avoids CLA error)
func (z *ZemuLedgerCosmosWrapper) GetVersion() (*ledger.VersionInfo, error) {
	z.version = ledger.VersionInfo{
		AppMode: 0,
		Major:   2,
		Minor:   1,
		Patch:   0,
	}
	return &z.version, nil
}

func (z *ZemuLedgerCosmosWrapper) Close() error {
	if z.api != nil {
		return z.api.Close()
	}
	return nil
}

func (z *ZemuLedgerCosmosWrapper) GetPublicKeySECP256K1(hdPath []uint32) ([]byte, error) {
	if z.api == nil {
		return nil, fmt.Errorf("no Speculos connection")
	}

	pathBytes := serializeHDPath(hdPath)
	cmd := []byte{0xE0, 0x02, 0x00, 0x00, byte(len(pathBytes))}
	cmd = append(cmd, pathBytes...)

	response, err := z.api.Exchange(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from Speculos Ethereum app: %w", err)
	}

	if len(response) < 2 {
		return nil, fmt.Errorf("invalid response length: %d", len(response))
	}

	pubKeyLen := int(response[0])
	if len(response) < (1 + pubKeyLen) {
		return nil, fmt.Errorf("response too short for pubkey length %d", pubKeyLen)
	}

	pubkey := response[1 : 1+pubKeyLen]
	return compressPublicKey(pubkey), nil
}

func (z *ZemuLedgerCosmosWrapper) GetAddressPubKeySECP256K1(hdPath []uint32, hrp string) ([]byte, string, error) {
	if z.api == nil {
		return nil, "", fmt.Errorf("no Speculos connection")
	}

	pathBytes := serializeHDPath(hdPath)
	cmd := []byte{0xE0, 0x02, 0x00, 0x00, byte(len(pathBytes))}
	cmd = append(cmd, pathBytes...)

	response, err := z.api.Exchange(cmd)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get address from Speculos Ethereum app: %w", err)
	}

	if len(response) < 2 {
		return nil, "", fmt.Errorf("invalid address response length: %d", len(response))
	}

	pubKeyLen := int(response[0])
	if len(response) < (1 + pubKeyLen + 1) {
		return nil, "", fmt.Errorf("response too short for pubkey")
	}

	pubkey := response[1 : 1+pubKeyLen]
	pubkey = compressPublicKey(pubkey)

	// Extract Ethereum address
	addrLenPos := 1 + int(response[0]) // Use original pubKeyLen from response
	if len(response) <= addrLenPos {
		return nil, "", fmt.Errorf("response too short for address length")
	}

	addrLen := int(response[addrLenPos])
	addrStartPos := addrLenPos + 1

	if len(response) < (addrStartPos + addrLen) {
		return nil, "", fmt.Errorf("response too short for address")
	}

	ethAddressHex := string(response[addrStartPos : addrStartPos+addrLen])
	bech32Address, err := convertEthAddressToBech32(ethAddressHex, hrp)
	if err != nil {
		return nil, "", fmt.Errorf("failed to convert ETH address to bech32: %w", err)
	}

	return pubkey, bech32Address, nil
}

func convertEthAddressToBech32(ethAddress, hrp string) (string, error) {
	if len(ethAddress) > 2 && ethAddress[:2] == "0x" {
		ethAddress = ethAddress[2:]
	}

	addressBytes, err := hex.DecodeString(ethAddress)
	if err != nil {
		return "", fmt.Errorf("invalid hex address: %w", err)
	}

	if len(addressBytes) != 20 {
		return "", fmt.Errorf("invalid address length: %d, expected 20", len(addressBytes))
	}

	bech32Addr, err := bech32.ConvertAndEncode(hrp, addressBytes)
	if err != nil {
		return "", fmt.Errorf("failed to convert and encode bech32: %w", err)
	}

	return bech32Addr, nil
}

func (z *ZemuLedgerCosmosWrapper) SignSECP256K1(hdPath []uint32, signDocBytes []byte, p2 byte) ([]byte, error) {
	if z.api == nil {
		return nil, fmt.Errorf("no Speculos connection")
	}

	// Create message hash for signing (Cosmos typically uses this)
	// The signDocBytes should be the transaction hash to sign
	msgToSign := signDocBytes
	if len(msgToSign) > 32 {
		// If message is longer than 32 bytes, take first 32 (or hash it)
		msgToSign = msgToSign[:32]
	}

	// Use Ethereum app personal sign: CLA=0xE0, INS=0x08
	pathBytes := serializeHDPath(hdPath)
	cmdData := make([]byte, 0, len(pathBytes)+4+len(msgToSign))
	cmdData = append(cmdData, pathBytes...)

	// Add message length (4 bytes, big endian)
	msgLen := len(msgToSign)
	cmdData = append(cmdData, byte(msgLen>>24), byte(msgLen>>16), byte(msgLen>>8), byte(msgLen))
	cmdData = append(cmdData, msgToSign...)

	cmd := []byte{0xE0, 0x08, 0x00, 0x00, byte(len(cmdData))} // INS=0x08 (personal sign)
	cmd = append(cmd, cmdData...)

	response, err := z.api.Exchange(cmd)
	if err != nil {
		// If personal sign fails, try the transaction sign approach
		return z.fallbackTransactionSign(hdPath, signDocBytes, p2)
	}

	// Parse signature response: v (1 byte) + r (32 bytes) + s (32 bytes) = 65 bytes total
	if len(response) < 65 {
		return nil, fmt.Errorf("signature response too short: %d bytes, expected 65", len(response))
	}

	// Return r + s (64 bytes), skip v for now
	signature := response[1:65] // Skip v, take r+s
	return signature, nil
}

// Fallback to transaction signing if personal sign doesn't work
func (z *ZemuLedgerCosmosWrapper) fallbackTransactionSign(hdPath []uint32, signDocBytes []byte, p2 byte) ([]byte, error) {
	// Use Ethereum app transaction sign: CLA=0xE0, INS=0x04
	pathBytes := serializeHDPath(hdPath)

	// For transaction signing, we need to format the data differently
	// This is a simplified approach - might need adjustment based on actual requirements
	cmdData := append(pathBytes, signDocBytes...)

	// Split if too large (max 255 bytes per APDU)
	if len(cmdData) > 200 {
		cmdData = cmdData[:200]
	}

	cmd := []byte{0xE0, 0x04, 0x00, 0x00, byte(len(cmdData))} // INS=0x04 (transaction sign)
	cmd = append(cmd, cmdData...)

	response, err := z.api.Exchange(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with Speculos Ethereum app (both methods): %w", err)
	}

	// Parse signature response
	if len(response) < 65 {
		return nil, fmt.Errorf("signature response too short: %d bytes, expected 65", len(response))
	}

	// Return r + s (64 bytes), skip v for now
	signature := response[1:65]
	return signature, nil
}

func findZemuCosmosAppSkipVersion() (*ZemuLedgerCosmosWrapper, error) {
	ledgerAdmin := ledger_go.NewLedgerAdmin()
	ledgerAPI, err := ledgerAdmin.Connect(0)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Speculos: %w", err)
	}
	wrapper := &ZemuLedgerCosmosWrapper{
		api: ledgerAPI,
		version: ledger.VersionInfo{
			AppMode: 0,
			Major:   2,
			Minor:   1,
			Patch:   0,
		},
	}
	return wrapper, nil
}

// ZemuEVMWrapper wraps the Zemu/Speculos device for EVM compatibility
type ZemuEVMWrapper struct {
	cosmosDevice *ZemuLedgerCosmosWrapper
}

func (w *ZemuEVMWrapper) Close() error {
	if w.cosmosDevice != nil {
		return w.cosmosDevice.Close()
	}
	return nil
}

func (w *ZemuEVMWrapper) GetPublicKeySECP256K1(hdPath []uint32) ([]byte, error) {
	if w.cosmosDevice == nil {
		return nil, fmt.Errorf("no Speculos device connected")
	}

	return w.cosmosDevice.GetPublicKeySECP256K1(hdPath)
}

func (w *ZemuEVMWrapper) GetAddressPubKeySECP256K1(hdPath []uint32, hrp string) ([]byte, string, error) {
	if w.cosmosDevice == nil {
		return nil, "", fmt.Errorf("no Speculos device connected")
	}
	return w.cosmosDevice.GetAddressPubKeySECP256K1(hdPath, hrp)
}

func (w *ZemuEVMWrapper) SignSECP256K1(hdPath []uint32, signDocBytes []byte, p2 byte) ([]byte, error) {
	if w.cosmosDevice == nil {
		return nil, fmt.Errorf("no Speculos device connected")
	}

	return w.cosmosDevice.SignSECP256K1(hdPath, signDocBytes, p2)
}
