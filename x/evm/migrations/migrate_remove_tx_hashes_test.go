package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
	"github.com/tabilabs/tabi-v3/x/evm/migrations"
	"github.com/tabilabs/tabi-v3/x/evm/types"
	tmtypes "github.com/tendermint/tendermint/proto/tendermint/types"
)

func TestRemoveTxHashes(t *testing.T) {
	k := testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.NewContext(false, tmtypes.Header{})
	store := ctx.KVStore(k.GetStoreKey())
	store.Set(types.TxHashesKey(1), []byte{1})
	store.Set(types.TxHashesKey(2), []byte{2})
	require.Equal(t, []byte{1}, store.Get(types.TxHashesKey(1)))
	require.Equal(t, []byte{2}, store.Get(types.TxHashesKey(2)))
	require.NoError(t, migrations.RemoveTxHashes(ctx, &k))
	require.Nil(t, store.Get(types.TxHashesKey(1)))
	require.Nil(t, store.Get(types.TxHashesKey(2)))
}
