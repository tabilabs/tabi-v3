package tools

import (
	"github.com/spf13/cobra"

	hasher "github.com/tabilabs/tabi-v3/tools/hash_verification/cmd"
	migration "github.com/tabilabs/tabi-v3/tools/migration/cmd"
	scanner "github.com/tabilabs/tabi-v3/tools/tx-scanner/cmd"
)

func ToolCmd() *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "A set of useful tools for tabi chain",
	}
	toolsCmd.AddCommand(scanner.ScanCmd())
	toolsCmd.AddCommand(migration.MigrateCmd())
	toolsCmd.AddCommand(migration.VerifyMigrationCmd())
	toolsCmd.AddCommand(migration.GenerateStats())
	toolsCmd.AddCommand(hasher.GenerateIavlHashCmd())
	toolsCmd.AddCommand(hasher.GeneratePebbleHashCmd())
	return toolsCmd
}
