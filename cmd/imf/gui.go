// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

// Package gui provides a local web-based GUI for IMF container management.
// It serves a single-page application and exposes REST API endpoints that
// wrap the container package operations.
package main

import (
	"archive/zip"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/immutable-container/imf/pkg/anchor"
	"github.com/immutable-container/imf/pkg/container"
	imfcrypto "github.com/immutable-container/imf/pkg/crypto"
)

// guiState holds the current working state for the GUI session.
type guiState struct {
	WorkDir    string // temporary working directory for this session
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	KeyLoaded  bool
}

var state guiState

// apiResponse is the standard JSON response envelope.
type apiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// runGUI starts a local web server that serves the IMF graphical interface.
// It creates a working directory on the user's Desktop for easy access to
// created .imf files. Falls back to a temp directory if Desktop is not found.
// Registers all REST API routes, finds an available port on localhost, and
// opens the user's default browser. All operations happen locally — the server
// only listens on 127.0.0.1 and never exposes data to the network.
func runGUI() {
	// Use the user's Desktop as the working directory so .imf files are
	// easy to find. Fall back to a temp directory if Desktop doesn't exist.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}

	desktopDir := filepath.Join(homeDir, "Desktop")
	if info, err := os.Stat(desktopDir); err != nil || !info.IsDir() {
		// No Desktop folder — try ~/Downloads, then fall back to temp.
		desktopDir = filepath.Join(homeDir, "Downloads")
		if info, err := os.Stat(desktopDir); err != nil || !info.IsDir() {
			desktopDir, _ = os.MkdirTemp("", "imf-gui-*")
		}
	}
	state.WorkDir = desktopDir
	fmt.Printf("IMF working directory: %s\n", state.WorkDir)
	fmt.Println("Created .imf files will appear here.")

	mux := http.NewServeMux()

	// Serve the single-page HTML application.
	mux.HandleFunc("/", handleIndex)

	// REST API endpoints for container operations.
	mux.HandleFunc("/api/keygen", handleKeygen)
	mux.HandleFunc("/api/key-status", handleKeyStatus)
	mux.HandleFunc("/api/load-key", handleLoadKey)
	mux.HandleFunc("/api/create", handleCreate)
	mux.HandleFunc("/api/add", handleAddFiles)
	mux.HandleFunc("/api/seal", handleSeal)
	mux.HandleFunc("/api/verify", handleVerify)
	mux.HandleFunc("/api/extract", handleExtract)
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/list", handleList)
	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/download-zip", handleDownloadZip)
	mux.HandleFunc("/api/browse", handleBrowse)
	mux.HandleFunc("/api/serve-file", handleServeFile)
	mux.HandleFunc("/api/upload-container", handleUploadContainer)
	mux.HandleFunc("/api/anchor", handleAnchor)
	mux.HandleFunc("/api/anchor-verify", handleAnchorVerify)
	mux.HandleFunc("/api/workdir", handleWorkDir)
	mux.HandleFunc("/api/export-key", handleExportKey)

	// Find an available port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding port: %v\n", err)
		os.Exit(1)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	fmt.Printf("IMF GUI running at %s\n", url)
	fmt.Println("Press Ctrl+C to stop")

	// Open the browser automatically.
	go openBrowser(url)

	// Start the server.
	http.Serve(listener, mux)
}

// openBrowser opens the default browser on the user's platform.
func openBrowser(url string) {
	time.Sleep(500 * time.Millisecond) // give server a moment to start
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}

// --- API Handlers ---

func handleKeygen(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	kp, err := imfcrypto.GenerateKeyPair()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	state.PrivateKey = kp.PrivateKey
	state.PublicKey = kp.PublicKey
	state.KeyLoaded = true

	// Keys stay in memory — no .pem files written to disk.
	// Users can export explicitly via /api/export-key if needed.

	jsonSuccess(w, "Key pair generated", nil)
}

// handleKeyStatus returns whether a signing key is currently loaded.
func handleKeyStatus(w http.ResponseWriter, r *http.Request) {
	jsonSuccess(w, "", map[string]bool{"loaded": state.KeyLoaded})
}

func handleLoadKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	file, _, err := r.FormFile("key")
	if err != nil {
		jsonError(w, "No key file provided", 400)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "Error reading key file", 500)
		return
	}

	// Try parsing as private key first, then public key.
	privKey, err := imfcrypto.ParsePrivateKeyPEM(data)
	if err == nil {
		state.PrivateKey = privKey
		state.PublicKey = privKey.Public().(ed25519.PublicKey)
		state.KeyLoaded = true
		jsonSuccess(w, "Private key loaded", nil)
		return
	}

	pubKey, err := imfcrypto.ParsePublicKeyPEM(data)
	if err == nil {
		state.PublicKey = pubKey
		state.PrivateKey = nil
		state.KeyLoaded = true
		jsonSuccess(w, "Public key loaded (verify only)", nil)
		return
	}

	jsonError(w, "Could not parse key file — must be an IMF PEM key", 400)
}

// handleCreate creates a new empty .imf container in the session's work directory.
// Accepts a "name" form field; defaults to "container" if omitted.
func handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		name = "container"
	}
	if !strings.HasSuffix(name, ".imf") {
		name += ".imf"
	}

	containerPath := filepath.Join(state.WorkDir, name)
	os.Remove(containerPath) // allow recreating

	if err := container.Create(containerPath); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	jsonSuccess(w, fmt.Sprintf("Created %s", name), map[string]string{
		"path": containerPath,
		"name": name,
	})
}

// handleAddFiles accepts multipart file uploads and adds them to the current container.
// Files are temporarily written to the work directory, then added to the container
// via the container.Add() library function, which records SHA-256 hashes in the manifest.
func handleAddFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	containerName := r.FormValue("container")
	if containerName == "" {
		jsonError(w, "No container specified", 400)
		return
	}
	containerPath := filepath.Join(state.WorkDir, containerName)

	// Parse the multipart form (up to 100MB).
	r.ParseMultipartForm(100 << 20)

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		jsonError(w, "No files provided", 400)
		return
	}

	// Save uploaded files to temp directory, then add to container.
	var tempPaths []string
	for _, fh := range files {
		src, err := fh.Open()
		if err != nil {
			jsonError(w, fmt.Sprintf("Error opening %s: %v", fh.Filename, err), 500)
			return
		}

		tmpPath := filepath.Join(state.WorkDir, "upload_"+fh.Filename)
		dst, err := os.Create(tmpPath)
		if err != nil {
			src.Close()
			jsonError(w, fmt.Sprintf("Error creating temp file: %v", err), 500)
			return
		}

		io.Copy(dst, src)
		src.Close()
		dst.Close()
		tempPaths = append(tempPaths, tmpPath)
	}

	if err := container.Add(containerPath, tempPaths); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	// Clean up temp upload files.
	for _, p := range tempPaths {
		os.Remove(p)
	}

	jsonSuccess(w, fmt.Sprintf("Added %d file(s)", len(files)), nil)
}

// handleSeal seals the container using the session's loaded private key.
// Accepts optional passphrase (for AES-256-GCM encryption), expiration date,
// and embed_key flag. Once sealed, the container becomes permanently immutable.
func handleSeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	containerName := r.FormValue("container")
	passphrase := r.FormValue("passphrase")
	expiresStr := r.FormValue("expires")
	embedKey := r.FormValue("embed_key") == "true"

	if containerName == "" {
		jsonError(w, "No container specified", 400)
		return
	}
	if state.PrivateKey == nil {
		jsonError(w, "No private key loaded — generate or load a key first", 400)
		return
	}

	containerPath := filepath.Join(state.WorkDir, containerName)

	opts := container.SealOptions{
		PrivateKey:  state.PrivateKey,
		EmbedPubKey: embedKey,
		Passphrase:  passphrase,
	}

	if expiresStr != "" {
		t, err := time.Parse("2006-01-02", expiresStr)
		if err != nil {
			jsonError(w, "Invalid date format (use YYYY-MM-DD)", 400)
			return
		}
		opts.ExpiresAt = &t
	}

	if err := container.Seal(containerPath, opts); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	jsonSuccess(w, "Container sealed", nil)
}

// handleVerify verifies a container's cryptographic integrity.
// Checks the Ed25519 signature and recomputes all file hashes.
// Accepts the container via multipart upload or by name in the work directory.
func handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	// Accept either a container name (in workdir) or an uploaded file.
	containerPath, err := resolveContainer(r)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	opts := container.VerifyOptions{
		IgnoreExpiry: r.FormValue("ignore_expiry") == "true",
	}

	if err := container.Verify(containerPath, opts); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	jsonSuccess(w, "Signature and integrity verified", nil)
}

// handleExtract extracts files from a sealed container into the work directory.
// If encrypted, the correct passphrase must be provided. Extracted files are
// accessible via the /api/browse and /api/download endpoints.
func handleExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	containerPath, err := resolveContainer(r)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	passphrase := r.FormValue("passphrase")
	outputDir := filepath.Join(state.WorkDir, "extracted")
	os.RemoveAll(outputDir)

	err = container.Extract(containerPath, container.ExtractOptions{
		Passphrase:   passphrase,
		IgnoreExpiry: r.FormValue("ignore_expiry") == "true",
		OutputDir:    outputDir,
	})
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	// List extracted files.
	var extractedFiles []string
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			extractedFiles = append(extractedFiles, info.Name())
		}
		return nil
	})

	jsonSuccess(w, fmt.Sprintf("Extracted %d file(s)", len(extractedFiles)), map[string]interface{}{
		"files":      extractedFiles,
		"output_dir": outputDir,
	})
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	containerPath, err := resolveContainer(r)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	info, err := container.GetInfo(containerPath)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	jsonSuccess(w, "", info)
}

func handleList(w http.ResponseWriter, r *http.Request) {
	containerPath, err := resolveContainer(r)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	files, err := container.ListFiles(containerPath)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	jsonSuccess(w, "", files)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		jsonError(w, "No file specified", 400)
		return
	}

	// Only allow downloads from our work directory.
	fullPath := filepath.Join(state.WorkDir, file)
	if !strings.HasPrefix(fullPath, state.WorkDir) {
		jsonError(w, "Invalid path", 400)
		return
	}

	// Check extracted directory too.
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fullPath = filepath.Join(state.WorkDir, "extracted", file)
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(fullPath)))
	http.ServeFile(w, r, fullPath)
}

// handleDownloadZip bundles all extracted files into a single ZIP for download.
// handleDownloadZip bundles all extracted files into a single ZIP archive for download.
// This provides a convenient way to download all files at once from the GUI.
func handleDownloadZip(w http.ResponseWriter, r *http.Request) {
	extractedDir := filepath.Join(state.WorkDir, "extracted")
	if _, err := os.Stat(extractedDir); os.IsNotExist(err) {
		jsonError(w, "No extracted files found", 404)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"extracted-files.zip\"")

	zw := zip.NewWriter(w)
	defer zw.Close()

	filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		f, err := zw.Create(info.Name())
		if err != nil {
			return nil
		}
		f.Write(data)
		return nil
	})
}

// fileDetail holds metadata for the file browser.
type fileDetail struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	Type     string `json:"type"`     // "image", "pdf", "text", "code", "document", "archive", "other"
	MimeType string `json:"mimeType"` // actual MIME type for preview
	Ext      string `json:"ext"`
}

// handleBrowse returns detailed file listing for the Finder-style browser.
// handleBrowse returns metadata for all extracted files (name, size, type, modified date).
// Powers the Finder-style file browser in the GUI's Extract panel.
func handleBrowse(w http.ResponseWriter, r *http.Request) {
	extractedDir := filepath.Join(state.WorkDir, "extracted")
	if _, err := os.Stat(extractedDir); os.IsNotExist(err) {
		jsonSuccess(w, "", []fileDetail{})
		return
	}

	var files []fileDetail
	filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		files = append(files, fileDetail{
			Name:     info.Name(),
			Size:     info.Size(),
			Modified: info.ModTime().Format("Jan 2, 2006 3:04 PM"),
			Type:     classifyFile(ext),
			MimeType: mimeForExt(ext),
			Ext:      ext,
		})
		return nil
	})

	jsonSuccess(w, "", files)
}

// handleServeFile serves a file inline for preview (not as download).
func handleServeFile(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "No file specified", 400)
		return
	}

	// Security: only serve from extracted directory.
	fullPath := filepath.Join(state.WorkDir, "extracted", filepath.Base(file))
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "File not found", 404)
		return
	}

	// Set content type for inline display.
	ext := strings.ToLower(filepath.Ext(file))
	mime := mimeForExt(ext)
	if mime != "" {
		w.Header().Set("Content-Type", mime)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(file)))
	http.ServeFile(w, r, fullPath)
}

// classifyFile returns a category based on file extension.
func classifyFile(ext string) string {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".ico":
		return "image"
	case ".pdf":
		return "pdf"
	case ".txt", ".md", ".csv", ".log", ".json", ".xml", ".yaml", ".yml", ".toml":
		return "text"
	case ".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".h", ".rs", ".rb", ".sh", ".html", ".css":
		return "code"
	case ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".odt", ".rtf":
		return "document"
	case ".zip", ".tar", ".gz", ".7z", ".rar", ".imf":
		return "archive"
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".m4a":
		return "audio"
	case ".mp4", ".mov", ".avi", ".mkv", ".webm":
		return "video"
	default:
		return "other"
	}
}

// mimeForExt returns a MIME type for common extensions.
func mimeForExt(ext string) string {
	mimes := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
		".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
		".pdf": "application/pdf", ".txt": "text/plain", ".md": "text/plain",
		".csv": "text/csv", ".json": "application/json", ".xml": "text/xml",
		".html": "text/html", ".css": "text/css", ".js": "text/javascript",
		".go": "text/plain", ".py": "text/plain", ".sh": "text/plain",
		".log": "text/plain", ".yaml": "text/plain", ".yml": "text/plain",
		".mp3": "audio/mpeg", ".wav": "audio/wav", ".mp4": "video/mp4",
	}
	if m, ok := mimes[ext]; ok {
		return m
	}
	return "application/octet-stream"
}

// handleUploadContainer saves an uploaded .imf file to the work directory
// so subsequent operations can reference it by name.
// handleUploadContainer accepts an .imf file upload and saves it to the work directory.
// Used when the user drags an existing container into the GUI for verification or extraction.
func handleUploadContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	file, header, err := r.FormFile("container_file")
	if err != nil {
		jsonError(w, "No container file provided", 400)
		return
	}
	defer file.Close()

	dstPath := filepath.Join(state.WorkDir, header.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		jsonError(w, fmt.Sprintf("Error saving container: %v", err), 500)
		return
	}
	io.Copy(dst, file)
	dst.Close()

	jsonSuccess(w, "Container uploaded", map[string]string{"path": dstPath})
}

// --- Helpers ---

// resolveContainer finds the container path from a form value or uploaded file.
// handleAnchor submits the container's SHA-256 hash to OpenTimestamps for
// blockchain anchoring. Returns the hash, proof path, and server used.
func handleAnchor(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	containerPath, err := resolveContainer(r)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	result, err := anchor.AnchorContainer(containerPath)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	jsonSuccess(w, "Anchored to Bitcoin", map[string]string{
		"hash":      result.ContainerHash,
		"proof":     result.ProofPath,
		"server":    result.Server,
		"timestamp": result.Timestamp.Format(time.RFC3339),
	})
}

// handleAnchorVerify checks that an existing .ots proof matches the container.
// Returns the hash and proof details if valid.
func handleAnchorVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", 405)
		return
	}

	containerPath, err := resolveContainer(r)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	result, err := anchor.VerifyAnchor(containerPath)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	jsonSuccess(w, "Anchor verified", map[string]interface{}{
		"hash":       result.ContainerHash,
		"proof_path": result.ProofPath,
		"proof_size": result.ProofSize,
		"matches":    result.HashMatches,
	})
}

// handleWorkDir returns the current working directory path so the GUI can
// show users where their .imf files are saved.
func handleWorkDir(w http.ResponseWriter, r *http.Request) {
	jsonSuccess(w, "", map[string]string{"path": state.WorkDir})
}

// handleExportKey downloads the private key as a .pem file.
// This is the only way keys leave memory — the user must explicitly request it.
func handleExportKey(w http.ResponseWriter, r *http.Request) {
	if state.PrivateKey == nil {
		http.Error(w, "No key to export", 400)
		return
	}
	pemData := imfcrypto.MarshalPrivateKeyPEM(state.PrivateKey)
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", "attachment; filename=\"imf_private.pem\"")
	w.Write(pemData)
}

// resolveContainer determines the container path from a request.
// It checks for a multipart file upload first, then falls back to a "container" form field
// referencing a file by name in the work directory.
func resolveContainer(r *http.Request) (string, error) {
	// Check for a named container in the work directory.
	name := r.FormValue("container")
	if name != "" {
		path := filepath.Join(state.WorkDir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Check for an uploaded container file.
	file, header, err := r.FormFile("container_file")
	if err == nil {
		defer file.Close()
		tmpPath := filepath.Join(state.WorkDir, header.Filename)
		dst, err := os.Create(tmpPath)
		if err != nil {
			return "", fmt.Errorf("saving uploaded container: %v", err)
		}
		io.Copy(dst, file)
		dst.Close()
		return tmpPath, nil
	}

	return "", fmt.Errorf("no container specified")
}

func jsonSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(apiResponse{
		Success: false,
		Error:   message,
	})
}
