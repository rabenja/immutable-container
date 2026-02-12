// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

// IMF Viewer is a Mac .app wrapper around the IMF GUI.
// When launched directly, it opens the GUI. When launched by double-clicking
// an .imf file, it opens the GUI with that container pre-loaded.
//
// This is the entry point for the "IMF Viewer.app" bundle.
package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	// Check if launched with a file argument (double-click on .imf file).
	// macOS passes the file path as the first argument when opening via
	// file association.
	var openFile string
	for _, arg := range os.Args[1:] {
		if strings.HasSuffix(arg, ".imf") {
			openFile = arg
			break
		}
	}

	// Find the imf binary — it lives next to this wrapper in the .app bundle.
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot determine executable path: %v\n", err)
		os.Exit(1)
	}
	bundleDir := filepath.Dir(execPath)
	imfBinary := filepath.Join(bundleDir, "imf")

	// Check if the imf binary exists.
	if _, err := os.Stat(imfBinary); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "IMF binary not found at %s\n", imfBinary)
		os.Exit(1)
	}

	if openFile != "" {
		// Launched with a file — start the GUI and tell it to open this file.
		launchWithFile(imfBinary, openFile)
	} else {
		// Launched directly — just start the GUI.
		cmd := exec.Command(imfBinary, "gui")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

// launchWithFile starts the GUI server and navigates to the container.
// It starts `imf gui` in the background, waits for the server to be ready,
// then uploads the container via the API.
func launchWithFile(imfBinary, filePath string) {
	// Start the GUI server.
	cmd := exec.Command(imfBinary, "gui")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()

	// Wait for the server to be ready by polling common ports.
	var serverURL string
	for i := 0; i < 50; i++ {
		for port := 52000; port < 52100; port++ {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
			if err == nil {
				conn.Close()
				serverURL = fmt.Sprintf("http://127.0.0.1:%d", port)
				break
			}
		}
		if serverURL != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if serverURL == "" {
		// Couldn't detect the server — GUI will still open, user can load manually.
		fmt.Fprintf(os.Stderr, "Could not detect GUI server port — GUI will open without the file pre-loaded\n")
		cmd.Wait()
		return
	}

	// Upload the container to the GUI via the API.
	absPath, _ := filepath.Abs(filePath)
	file, err := os.Open(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open file: %v\n", err)
		cmd.Wait()
		return
	}
	defer file.Close()

	// Copy the .imf file to the GUI's work directory by using the upload endpoint.
	// We use multipart form upload.
	uploadContainer(serverURL, absPath)

	// Open the browser to the server URL.
	if runtime.GOOS == "darwin" {
		exec.Command("open", serverURL).Start()
	}

	cmd.Wait()
}

// uploadContainer copies an .imf file to the GUI via the upload API.
func uploadContainer(serverURL, filePath string) {
	uploadURL := serverURL + "/api/upload-container"

	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	// Create multipart request.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("container", filepath.Base(filePath))
	io.Copy(part, file)
	writer.Close()

	req, _ := http.NewRequest("POST", uploadURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	http.DefaultClient.Do(req)
}
