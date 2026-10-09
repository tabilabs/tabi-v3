package app

import (
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestMultisigStructure(t *testing.T) {
	leaf := &signing.SingleSignatureData{SignMode: signing.SignMode_SIGN_MODE_DIRECT, Signature: []byte("signature")}
	valid := multisig.NewMultisig(8)
	multisig.AddSignature(valid, leaf, 7)
	mode, validRaw := authtx.SignatureDataToModeInfoAndSig(valid)
	body, err := proto.Marshal(&txtypes.TxBody{})
	require.NoError(t, err)
	authInfo, err := proto.Marshal(&txtypes.AuthInfo{
		Fee:         &txtypes.Fee{GasLimit: 200000},
		SignerInfos: []*txtypes.SignerInfo{{ModeInfo: mode}},
	})
	require.NoError(t, err)
	emptyMulti, err := (&cryptotypes.MultiSignature{}).Marshal()
	require.NoError(t, err)
	for _, tc := range []struct {
		name       string
		signatures [][]byte
		wantError  bool
	}{
		{"missing outer signature", nil, true},
		{"missing inner signature", [][]byte{emptyMulti}, true},
		{"valid multisig", [][]byte{validRaw}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := proto.Marshal(&txtypes.TxRaw{BodyBytes: body, AuthInfoBytes: authInfo, Signatures: tc.signatures})
			require.NoError(t, err)
			decoded, err := MakeEncodingConfig().TxConfig.TxDecoder()(wire)
			require.NoError(t, err)
			require.NotPanics(t, func() {
				sigs, err := decoded.(authsigning.SigVerifiableTx).GetSignaturesV2()
				if tc.wantError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Len(t, sigs, 1)
					require.Equal(t, valid, sigs[0].Data)
				}
			})
		})
	}
}
