package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tabilabs/tabi-v3/testutil/keeper"
	"github.com/tabilabs/tabi-v3/x/evm/migrations"
	tmtypes "github.com/tendermint/tendermint/proto/tendermint/types"
)

func TestMigrateDisableRegisterPointer(t *testing.T) {
	k := testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.NewContext(false, tmtypes.Header{})
	migrations.MigrateDisableRegisterPointer(ctx, &k)
	require.NotPanics(t, func() { k.GetParams(ctx) })
}
