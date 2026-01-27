package evmrpc

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestRecoverSenderWithFallback(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	expectedFrom := crypto.PubkeyToAddress(privateKey.PublicKey)

	signWithSigner := func(key *ecdsa.PrivateKey, signer ethtypes.Signer) *ethtypes.Transaction {
		tx := ethtypes.NewTransaction(0, common.HexToAddress("0x1234"), big.NewInt(1000), 21000, big.NewInt(1), nil)
		signedTx, err := ethtypes.SignTx(tx, signer, key)
		if err != nil {
			t.Fatalf("SignTx failed: %v", err)
		}
		return signedTx
	}

	tests := []struct {
		name          string
		signer        ethtypes.Signer
		defaultSigner ethtypes.Signer
		wantFrom      common.Address
		wantErr       bool
	}{
		{
			name:          "unprotected legacy tx with EIP155 signer should fallback to Homestead",
			signer:        ethtypes.HomesteadSigner{},
			defaultSigner: ethtypes.NewEIP155Signer(big.NewInt(1)),
			wantFrom:      expectedFrom,
			wantErr:       false,
		},
		{
			name:          "unprotected legacy tx with London signer should fallback to Homestead",
			signer:        ethtypes.HomesteadSigner{},
			defaultSigner: ethtypes.NewLondonSigner(big.NewInt(1)),
			wantFrom:      expectedFrom,
			wantErr:       false,
		},
		{
			name:          "protected legacy tx should work with matching signer",
			signer:        ethtypes.NewEIP155Signer(big.NewInt(1)),
			defaultSigner: ethtypes.NewEIP155Signer(big.NewInt(1)),
			wantFrom:      expectedFrom,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := signWithSigner(privateKey, tt.signer)
			from, err := recoverSenderWithFallback(tx, tt.defaultSigner)
			if (err != nil) != tt.wantErr {
				t.Errorf("recoverSenderWithFallback() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && from != tt.wantFrom {
				t.Errorf("recoverSenderWithFallback() from = %v, want %v", from.Hex(), tt.wantFrom.Hex())
			}
		})
	}
}
