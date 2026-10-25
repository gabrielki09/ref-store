package tests

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"ref-store/helpers"
	"strings"
	"sync"
	"testing"
)

func TestSaveRestoredFile_CreatesFile(t *testing.T) {
	testTempDir := t.TempDir()

	t.Log("Diretório temporário de teste:", "testTempDir", testTempDir)

	outputPath := filepath.Join(testTempDir, "file.txt")
	expectedContent := []byte("ABCDWXYZABCD")

	if err := helpers.SaveRestoredFile(outputPath, expectedContent); err != nil {
		t.Fatalf("Erro ao salvar arquivo restaurado: %v", err)
	}

	contentRead, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Erro ao ler o arquivo restaurado: %v", err)
	}

	if !bytes.Equal(contentRead, expectedContent) {
		t.Fatalf(
			"Conteúdo diferente: esperado=%q, recebido=%q",
			expectedContent,
			contentRead,
		)
	}

	entries, err := os.ReadDir(testTempDir)
	if err != nil {
		t.Fatalf("Erro ao listar diretório temporário: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Esperava exatamente 1 arquivo, encontrei %d", len(entries))
	}

	if entries[0].Name() != "file.txt" {
		t.Fatalf("Arquivo inesperado: %s", entries[0].Name())
	}
}

func TestSaveRestoredFile_ExistingSameContent(t *testing.T) {
	testTempDir := t.TempDir()

	t.Log("Diretório temporário de teste:", "testTempDir", testTempDir)

	outputPath := filepath.Join(testTempDir, "file.txt")
	expectedContent := []byte("ABCDWXYZABCD")

	if err := helpers.SaveRestoredFile(outputPath, expectedContent); err != nil {
		t.Fatalf("Erro ao salvar o primeiro arquivo restaurado: %v", err)
	}

	if err := helpers.SaveRestoredFile(outputPath, expectedContent); err != nil {
		t.Fatalf("Erro ao salvar o segundo arquivo restaurado: %v", err)
	}

	contentRead, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Erro ao ler o arquivo restaurado: %v", err)
	}

	if !bytes.Equal(contentRead, expectedContent) {
		t.Fatalf(
			"Conteúdo diferente: esperado=%q, recebido=%q",
			expectedContent,
			contentRead,
		)
	}
}

func TestSaveRestoredFile_ExistingDifferentContent(t *testing.T) {
	testTempDir := t.TempDir()

	t.Log("Diretório temporário de teste:", "testTempDir", testTempDir)

	outputPath := filepath.Join(testTempDir, "file.txt")
	originalContent := []byte("ABCD")
	differentContent := []byte("WXYZ")

	if err := helpers.SaveRestoredFile(outputPath, originalContent); err != nil {
		t.Fatalf("Primeiro salvamento não esperava erro")
	}

	if err := helpers.SaveRestoredFile(outputPath, differentContent); err == nil {
		t.Fatalf("Salvamento esperava erro")
	}

	contentRead, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Erro ao ler o arquivo restaurado: %v", err)
	}

	if !bytes.Equal(contentRead, originalContent) {
		t.Fatalf(
			"Conteúdo diferente: esperado=%q, recebido=%q",
			originalContent,
			contentRead,
		)
	}

	entries, err := os.ReadDir(testTempDir)
	if err != nil {
		t.Fatalf("Erro ao listar diretório temporário: %v", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "temp-") {
			t.Fatalf("Arquivo temporário não removido: %s", entry.Name())
		}
	}
}

func TestSaveRestoredFile_InvalidDirectory(t *testing.T) {
	testTempDir := t.TempDir()

	t.Log("Diretório temporário de teste:", "testTempDir", testTempDir)

	outputPath := filepath.Join(testTempDir, "pasta-inexistente", "file.txt")
	expectedContent := []byte("ABCDWXYZABCD")

	err := helpers.SaveRestoredFile(outputPath, expectedContent)

	if err == nil {
		t.Fatalf("Esperava erro ao acessar diretório não existente")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Esperava os.ErrNotExist, mas recebeu: %v", err)
	}
}

func TestSaveRestoredFile_ConcurrentDifferentContent(t *testing.T) {
	var wg sync.WaitGroup

	testTempDir := t.TempDir()

	t.Log("Diretório temporário de teste:", "testTempDir", testTempDir)

	outputPath := filepath.Join(testTempDir, "file.txt")

	start := make(chan struct{})
	results := make(chan error, 2)

	contents := [][]byte{
		[]byte("ABCD"),
		[]byte("WXYZ"),
	}

	for _, content := range contents {
		wg.Go(func() {
			<-start
			results <- helpers.SaveRestoredFile(outputPath, content)
		})
	}
	close(start)

	wg.Wait()

	close(results)

	contentRead, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Erro ao ler o arquivo restaurado: %v", err)
	}

	if !bytes.Equal(contentRead, contents[0]) {
		if !bytes.Equal(contentRead, contents[1]) {

			t.Fatalf(
				"Conteúdo inesperado: %q",
				contentRead,
			)
		}
	}

	entries, err := os.ReadDir(testTempDir)
	if err != nil {
		t.Fatalf("Erro ao listar diretório temporário: %v", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "temp-") {
			t.Fatalf("Arquivo temporário não removido: %s", entry.Name())
		}
	}

	successCount := 0
	errorCount := 0

	for err := range results {
		if err != nil {
			errorCount += 1
		} else {
			successCount += 1
		}
	}

	if successCount != 1 || errorCount != 1 {
		t.Fatalf(
			"Esperava 1 sucesso e 1 erro, recebi %d sucessos e %d erros",
			successCount,
			errorCount,
		)
	}
}
