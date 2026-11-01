package tests

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"ref-store/pkg"
	"strings"
	"sync"
	"testing"
)

type restoreFixture struct {
	rootDir     string
	storageDir  string
	manifestDir string
	restoredDir string
}

func newRestoreFixture(t *testing.T) restoreFixture {
	t.Helper()

	rootDir := t.TempDir()

	fixture := restoreFixture{
		rootDir:     rootDir,
		storageDir:  filepath.Join(rootDir, "storage"),
		manifestDir: filepath.Join(rootDir, "manifests"),
		restoredDir: filepath.Join(rootDir, "restored"),
	}

	for _, dir := range []string{
		fixture.storageDir,
		fixture.manifestDir,
		fixture.restoredDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Erro ao criar diretório %s: %v", dir, err)
		}
	}

	return fixture
}

func (f restoreFixture) createManifest(
	t *testing.T,
	sourceDir string,
	fileName string,
	content []byte,
) string {
	t.Helper()

	inputDir := filepath.Join(f.rootDir, sourceDir)

	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatalf("Erro ao criar diretório de entrada: %v", err)
	}

	inputPath := filepath.Join(inputDir, fileName)

	if err := os.WriteFile(inputPath, content, 0o600); err != nil {
		t.Fatalf("Erro ao criar arquivo original: %v", err)
	}

	manifestID, err := pkg.Store(
		inputPath,
		f.storageDir,
		f.manifestDir,
	)
	if err != nil {
		t.Fatalf("Erro ao armazenar arquivo: %v", err)
	}

	return filepath.Join(f.manifestDir, manifestID)
}

func assertFileContent(
	t *testing.T,
	filePath string,
	expected []byte,
) {
	t.Helper()

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Erro ao ler arquivo restaurado: %v", err)
	}

	if !bytes.Equal(content, expected) {
		t.Fatalf(
			"Conteúdo diferente: esperado=%q, recebido=%q",
			expected,
			content,
		)
	}
}

func assertNoTemporaryFiles(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Erro ao listar diretório: %v", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "temp-") {
			t.Fatalf(
				"Arquivo temporário não removido: %s",
				entry.Name(),
			)
		}
	}
}

func TestRestore_CreatesFile(t *testing.T) {
	f := newRestoreFixture(t)

	expectedContent := []byte("ABCDWXYZABCD")

	manifestPath := f.createManifest(
		t, "input", "file.txt", expectedContent,
	)

	err := pkg.Restore(
		manifestPath,
		f.storageDir,
		f.restoredDir,
	)
	if err != nil {
		t.Fatalf("Erro ao restaurar arquivo: %v", err)
	}

	outputPath := filepath.Join(f.restoredDir, "file.txt")

	assertFileContent(t, outputPath, expectedContent)
	assertNoTemporaryFiles(t, f.restoredDir)

	entries, err := os.ReadDir(f.restoredDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Fatalf(
			"Esperava exatamente 1 arquivo, encontrei %d",
			len(entries),
		)
	}
}

func TestRestore_ExistingSameContent(t *testing.T) {
	f := newRestoreFixture(t)

	expectedContent := []byte("ABCDWXYZABCD")

	manifestPath := f.createManifest(
		t, "input", "file.txt", expectedContent,
	)

	for i := 0; i < 2; i++ {
		if err := pkg.Restore(
			manifestPath,
			f.storageDir,
			f.restoredDir,
		); err != nil {
			t.Fatalf(
				"Restauração %d falhou: %v",
				i+1,
				err,
			)
		}
	}

	outputPath := filepath.Join(f.restoredDir, "file.txt")

	assertFileContent(t, outputPath, expectedContent)
	assertNoTemporaryFiles(t, f.restoredDir)
}

func TestRestore_ExistingDifferentContent(t *testing.T) {
	f := newRestoreFixture(t)

	originalContent := []byte("ABCD")
	differentContent := []byte("WXYZ")

	manifestPath := f.createManifest(
		t, "input", "file.txt", differentContent,
	)

	outputPath := filepath.Join(f.restoredDir, "file.txt")

	if err := os.WriteFile(
		outputPath,
		originalContent,
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	err := pkg.Restore(
		manifestPath,
		f.storageDir,
		f.restoredDir,
	)

	if err == nil {
		t.Fatal("Esperava erro ao restaurar conteúdo diferente")
	}

	// O arquivo anterior não pode ser alterado.
	assertFileContent(t, outputPath, originalContent)
	assertNoTemporaryFiles(t, f.restoredDir)
}

func TestRestore_InvalidDirectory(t *testing.T) {
	f := newRestoreFixture(t)

	expectedContent := []byte("ABCDWXYZABCD")

	manifestPath := f.createManifest(
		t, "input", "file.txt", expectedContent,
	)

	invalidDir := filepath.Join(
		f.rootDir,
		"pasta-inexistente",
		"restored",
	)

	err := pkg.Restore(
		manifestPath,
		f.storageDir,
		invalidDir,
	)

	if err == nil {
		t.Fatal("Esperava erro ao acessar diretório inexistente")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf(
			"Esperava os.ErrNotExist, recebeu: %v",
			err,
		)
	}
}

func TestRestore_ConcurrentDifferentContent(t *testing.T) {
	f := newRestoreFixture(t)

	contentA := []byte("ABCD")
	contentB := []byte("WXYZ")

	// Mesmo nome de arquivo em diretórios diferentes.
	// Os manifestos terão conteúdos distintos.
	manifestA := f.createManifest(
		t, "input-a", "file.txt", contentA,
	)

	manifestB := f.createManifest(
		t, "input-b", "file.txt", contentB,
	)

	var wg sync.WaitGroup

	start := make(chan struct{})
	results := make(chan error, 2)

	for _, manifestPath := range []string{
		manifestA,
		manifestB,
	} {
		wg.Add(1)

		go func(path string) {
			defer wg.Done()

			<-start

			results <- pkg.Restore(
				path,
				f.storageDir,
				f.restoredDir,
			)
		}(manifestPath)
	}

	close(start)

	wg.Wait()
	close(results)

	successCount := 0
	errorCount := 0

	for err := range results {
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	if successCount != 1 || errorCount != 1 {
		t.Fatalf(
			"Esperava 1 sucesso e 1 erro, recebi %d sucessos e %d erros",
			successCount,
			errorCount,
		)
	}

	outputPath := filepath.Join(f.restoredDir, "file.txt")

	contentRead, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Erro ao ler arquivo restaurado: %v", err)
	}

	if !bytes.Equal(contentRead, contentA) &&
		!bytes.Equal(contentRead, contentB) {
		t.Fatalf(
			"Conteúdo inesperado: %q",
			contentRead,
		)
	}

	assertNoTemporaryFiles(t, f.restoredDir)
}
