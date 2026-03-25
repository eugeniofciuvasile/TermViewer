package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileInfo struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

type ActiveUpload struct {
	File *os.File
	Path string
}

var activeUploads sync.Map // Map[string]*ActiveUpload

func listFiles(path string) (string, []FileInfo, error) {
	if path == "" || path == "~" {
		home, _ := os.UserHomeDir()
		path = home
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return absPath, nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	return absPath, files, nil
}

func streamFileToClient(path string, clientID string, writeJSON func(v interface{}) error) {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("Failed to open file for download", "path", path, "error", err)
		return
	}
	defer file.Close()

	filename := filepath.Base(path)
	transferID := filename + "-" + time.Now().Format("150405")
	buffer := make([]byte, 32*1024) // 32KB chunks
	chunkIndex := 0

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			isLast := (err == io.EOF)
			errWrite := writeJSON(map[string]interface{}{
				"type": "file_data",
				"payload": map[string]interface{}{
					"transfer_id": transferID,
					"filename":    filename,
					"chunk_index": chunkIndex,
					"is_last":     isLast,
					"data":        base64.StdEncoding.EncodeToString(buffer[:n]),
				},
			})
			if errWrite != nil {
				return
			}
			chunkIndex++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("Error reading file during transfer", "path", path, "error", err)
			break
		}
	}
	slog.Info("File transfer completed", "path", path, "client_id", clientID)
}

func handleFileUploadStart(transferID, filename, destPath string) error {
	fullPath := filepath.Join(destPath, filename)
	
	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	activeUploads.Store(transferID, &ActiveUpload{
		File: f,
		Path: fullPath,
	})
	slog.Info("Starting file upload to host", "path", fullPath, "transfer_id", transferID)
	return nil
}

func handleIncomingFileData(transferID string, dataBase64 string, isLast bool) error {
	val, exists := activeUploads.Load(transferID)
	if !exists {
		return fmt.Errorf("transfer ID not found: %s", transferID)
	}
	upload := val.(*ActiveUpload)

	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return err
	}

	_, err = upload.File.Write(data)
	if err != nil {
		return err
	}

	if isLast {
		upload.File.Close()
		activeUploads.Delete(transferID)
		slog.Info("File upload completed", "path", upload.Path, "transfer_id", transferID)
	}
	return nil
}
