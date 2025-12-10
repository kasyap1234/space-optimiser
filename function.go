package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

//go:embed static/*
var staticFiles embed.FS

// In-memory storage for visualizations
var (
	visualizations sync.Map // map[string]string - ID to HTML content
)

// PackRequest defines the input structure
type PackRequest struct {
	Items []InputItem `json:"items"`
	Boxes []InputBox  `json:"boxes"`
}

// PackResponse defines the output structure
type PackResponse struct {
	PackedBoxes      []PackedBox `json:"packed_boxes"`
	UnpackedItems    []InputItem `json:"unpacked_items"`
	TotalVolume      int         `json:"total_volume"`
	Utilization      float64     `json:"utilization_percent"`
	VisualizationURL string      `json:"visualization_url,omitempty"`
}

// Packer is the HTTP handler entry point used by Cloud Run
func Packer(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Route handling
	if r.URL.Path == "/pack" && r.Method == http.MethodPost {
		handlePack(w, r)
		return
	}

	// Visualization endpoint
	if strings.HasPrefix(r.URL.Path, "/visualize/") {
		handleVisualization(w, r)
		return
	}

	handleStatic(w, r)
}

func handlePack(w http.ResponseWriter, r *http.Request) {
	var req PackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 || len(req.Boxes) == 0 {
		http.Error(w, "Items and Boxes are required", http.StatusBadRequest)
		return
	}

	// Run packing logic
	packedBoxes, unpackedItems := Pack(req.Items, req.Boxes)

	// Calculate stats
	totalBoxVolume := 0
	totalItemVolume := 0

	for _, box := range packedBoxes {
		// Find box dims from request (inefficient but safe)
		var b InputBox
		for _, ib := range req.Boxes {
			if ib.ID == box.BoxID {
				b = ib
				break
			}
		}
		totalBoxVolume += b.W * b.H * b.D

		for _, item := range box.Contents {
			totalItemVolume += item.W * item.H * item.D
		}
	}

	utilization := 0.0
	if totalBoxVolume > 0 {
		utilization = (float64(totalItemVolume) / float64(totalBoxVolume)) * 100
	}

	// Generate visualization
	vizID := uuid.New().String()
	vizData := VisualizationData{
		PackedBoxes: packedBoxes,
		Boxes:       req.Boxes,
		RequestID:   vizID,
	}

	vizHTML, err := GenerateVisualizationHTML(vizData)
	if err == nil {
		// Store visualization in memory
		visualizations.Store(vizID, vizHTML)
	}

	// Build visualization URL
	vizURL := ""
	if err == nil {
		// Get the host from the request
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		host := r.Host
		if host == "" {
			host = "localhost:8080"
		}
		vizURL = scheme + "://" + host + "/visualize/" + vizID
	}

	resp := PackResponse{
		PackedBoxes:      packedBoxes,
		UnpackedItems:    unpackedItems,
		TotalVolume:      totalBoxVolume,
		Utilization:      utilization,
		VisualizationURL: vizURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleVisualization(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid visualization ID", http.StatusBadRequest)
		return
	}

	vizID := parts[2]

	// Retrieve visualization from storage
	htmlContent, ok := visualizations.Load(vizID)
	if !ok {
		http.Error(w, "Visualization not found or expired", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent.(string)))
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	// Create a sub-filesystem for the "static" directory
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.FileServer(http.FS(fsys)).ServeHTTP(w, r)
}
