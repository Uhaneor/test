package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/pbkdf2"
)

// settings for Phantom Wallet Vaults

// PhantomVault struct
type Vault struct {
	EncryptedData []byte
	Salt          []byte
	Nonce         []byte
	Iterations    int
	Decrypted     bool
}

// isValid function as placeholder, always returning true
func isValid(s []byte) bool {
	return true
}

// decryptVault using secretbox
func decryptVault(encryptedData, password, salt, nonce []byte, iterations int) ([]byte, error) {
	if len(nonce) != 24 {
		return nil, fmt.Errorf("nonce must be exactly 24 bytes long")
	}

	key := pbkdf2.Key(password, salt, iterations, 32, sha256.New)

	var nonceArray [24]byte
	copy(nonceArray[:], nonce)
	var keyArray [32]byte
	copy(keyArray[:], key)

	decrypted, ok := secretbox.Open(nil, encryptedData, &nonceArray, &keyArray)
	if !ok {
		return nil, fmt.Errorf("decryption failed")
	}

	return decrypted, nil
}

// parse Phantom vault
func readVaultData(filePath string) ([]Vault, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var vaults []Vault
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var hash struct {
			EncryptedKey struct {
				Digest     string `json:"digest"`
				Encrypted  string `json:"encrypted"`
				Salt       string `json:"salt"`
				Nonce      string `json:"nonce"`
				Iterations int    `json:"iterations"`
				Kdf        string `json:"kdf"`
			} `json:"encryptedKey"`
		}

		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &hash); err != nil {
			log.Printf("Error parsing JSON: %v\n", err)
			continue
		}

		// sanity checks for Phanton vault
		if hash.EncryptedKey.Digest != "sha256" ||
			hash.EncryptedKey.Kdf != "pbkdf2" ||
			hash.EncryptedKey.Iterations <= 0 ||
			len(hash.EncryptedKey.Encrypted) == 0 ||
			len(hash.EncryptedKey.Salt) == 0 ||
			len(hash.EncryptedKey.Nonce) == 0 {
			log.Printf("Invalid or incomplete data encountered in JSON: %v\n", line)
			continue
		}

		encryptedData := base58.Decode(hash.EncryptedKey.Encrypted)
		salt := base58.Decode(hash.EncryptedKey.Salt)
		nonce := base58.Decode(hash.EncryptedKey.Nonce)

		if len(encryptedData) == 0 || len(salt) == 0 || len(nonce) == 0 {
			log.Printf("Error decoding base58 data: possibly incorrect format or content: %v\n", line)
			continue
		}

		vault := Vault{
			EncryptedData: encryptedData,
			Salt:          salt,
			Nonce:         nonce,
			Iterations:    hash.EncryptedKey.Iterations,
		}
		vaults = append(vaults, vault)
	}

	return vaults, nil
}
