package app

import "testing"

func TestReadGenesisImportConfigRejectsInvalidImportFile(t *testing.T) {
	if _, err := ReadGenesisImportConfig(mapAppOptions{flagGenesisImportFile: []string{"genesis.json"}}); err == nil {
		t.Fatal("expected invalid import file type to return an error")
	}
}
