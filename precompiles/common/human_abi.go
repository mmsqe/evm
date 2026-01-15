package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// ParseHumanReadableABI converts human-readable ABI signatures to an abi.ABI.
// For struct-like types, use inline tuple syntax: tuple(address addr, uint256 amount)
func ParseHumanReadableABI(signatures []string) (abi.ABI, error) {
	var abiElements []abi.ABIMarshaling

	for _, line := range signatures {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip struct definitions - use inline tuple syntax instead
		if strings.HasPrefix(line, "struct ") {
			continue
		}

		element, err := abi.ParseHumanReadableABI(line)
		if err != nil {
			return abi.ABI{}, fmt.Errorf("failed to parse signature '%s': %v", line, err)
		}

		if element != nil {
			abiElements = append(abiElements, element)
		}
	}

	jsonBytes, err := json.Marshal(abiElements)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to marshal ABI to JSON: %v", err)
	}

	parsedABI, err := abi.JSON(bytes.NewReader(jsonBytes))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse JSON ABI: %v", err)
	}

	return parsedABI, nil
}
