package main

import (
	"flag"
	"os"
	"path/filepath"
	"ref-store/logger"
	"ref-store/pkg"
	"ref-store/pkg/helpers"

	"charm.land/log/v2"
)

func main() {
	var (
		storeProcess   = flag.String("store", "", "Informa o caminho do arquivo que será armazenado")
		restoreProcess = flag.String("restore", "", "Informa o caminho do arquivo que será restaurado")

		deleteAllFiles = flag.Bool("delete", false, "Delete todos os arquivos gerados")

		debugLevel = flag.Bool("debug", false, "Habilita logs de debug")
	)

	flag.Parse()

	if *debugLevel {
		log.SetLevel(log.DebugLevel)
	}

	hasStore := *storeProcess != ""
	hasRestore := *restoreProcess != ""

	if (hasStore && hasRestore) ||
		(*deleteAllFiles && (hasStore || hasRestore)) ||
		(!hasStore && !hasRestore && !*deleteAllFiles) {

		logger.Error("Informe apenas uma operação: -store, -restore ou -delete")
		flag.Usage()
		os.Exit(2)
	}

	storageDir, manifestDir, err := helpers.CreateInitFolders()

	if err != nil {
		logger.Error("Erro ao criar os diretórios inicias:", "err", err)
		os.Exit(1)
	}

	if hasStore {
		savedManifestDir, err := pkg.Store(*storeProcess, storageDir, manifestDir)

		if err != nil {
			logger.Error("Erro ao salvar os dados do arquivo:", "err", err)
			os.Exit(1)
		}

		logger.Info("Caminho do manifesto gerado:", "manifest", savedManifestDir)
	}

	if hasRestore {
		restoredDir, err := helpers.CreateInformedFolder(helpers.RestoredDir)
		if err != nil {
			logger.Error("Erro ao criar diretório de armazenamento de arquivos restaurados:", "err", err)
			os.Exit(1)
		}

		manifestPath := filepath.Join(
			manifestDir,
			*restoreProcess,
		)

		if err := pkg.Restore(manifestPath, storageDir, restoredDir); err != nil {
			logger.Error("Erro ao restaurar arquivo:", "err", err)
			os.Exit(1)
		}
	}

	if *deleteAllFiles {
		if err := os.RemoveAll(storageDir); err != nil {
			logger.Error("Erro ao remover os arquivos do diretório: ", "storageDir", storageDir, "err", err)
			os.Exit(1)
		}

		if err := os.RemoveAll(manifestDir); err != nil {
			logger.Error("Erro ao remover os arquivos do diretório: ", "manifestDir", manifestDir, "err", err)
			os.Exit(1)
		}

		if err := os.RemoveAll("restored"); err != nil {
			logger.Error("Erro ao remover os arquivos do diretório: ", "restored", "restored", "err", err)
			os.Exit(1)
		}

		if _, _, err := helpers.CreateInitFolders("restored"); err != nil {
			logger.Error("Erro ao criar os diretórios inicias:", "err", err)
			os.Exit(1)
		}

		logger.Info("Diretórios limpos e recriados com sucesso.")
	}
}
