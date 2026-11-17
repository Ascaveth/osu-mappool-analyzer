package beatmapimport

import (
	"context"
	"errors"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage/memory"
	"os"
	"testing"
)

func TestImportAndReimport(t *testing.T) {
	source, err := os.ReadFile("../osufile/testdata/sample.osu")
	if err != nil {
		t.Fatal(err)
	}
	repo := memory.NewBeatmapRepository()
	first, isNew, err := Import(context.Background(), repo, nil, source)
	if err != nil || !isNew {
		t.Fatalf("first import: new=%v err=%v", isNew, err)
	}
	second, isNew, err := Import(context.Background(), repo, nil, source)
	if err != nil || isNew || first.ID != second.ID {
		t.Fatalf("reimport: new=%v err=%v", isNew, err)
	}
}

func TestInvalidFile(t *testing.T) {
	_, _, err := Import(context.Background(), memory.NewBeatmapRepository(), nil, []byte("invalid"))
	var invalid *InvalidFileError
	if !errors.As(err, &invalid) {
		t.Fatalf("want invalid file error, got %v", err)
	}
}
