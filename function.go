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

// visualizations stores visualization HTML content by ID.
var visualizations sync.Map

// PackRequest defines the input structure for the packing API.
type PackRequest struct {
	Items []InputItem `json:"items"`
	Boxes []InputBox  `json:"boxes"`
}

// PackResponse defines the output structure for the packing API.
type PackResponse struct {
	PackedBoxes      []PackedBox `json:"packed_boxes"`
	UnpackedItems    []InputItem `json:"unpacked_items"`
	TotalVolume      int         `json:"total_volume"`
	Utilization      float64     `json:"utilization_percent"`
	VisualizationURL string      `json:"visualization_url,omitempty"`
}

// Packer is the HTTP handler entry point.
func Packer(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch {
	case r.URL.Path == "/pack" && r.Method == http.MethodPost:
		handlePack(w, r)
	case strings.HasPrefix(r.URL.Path, "/visualize/"):
		handleVisualization(w, r)
	default:
		handleStatic(w, r)
	}
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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

	packedBoxes, unpackedItems := Pack(req.Items, req.Boxes)

	boxByID := make(map[string]InputBox, len(req.Boxes))
	for _, b := range req.Boxes {
		boxByID[b.ID] = b
	}

	var totalBoxVolume, totalItemVolume int
	for _, box := range packedBoxes {
		b := boxByID[box.BoxID]
		totalBoxVolume += b.W * b.H * b.D
		for _, item := range box.Contents {
			totalItemVolume += item.W * item.H * item.D
		}
	}

	var utilization float64
	if totalBoxVolume > 0 {
		utilization = float64(totalItemVolume) / float64(totalBoxVolume) * 100
	}

	vizID := uuid.New().String()
	vizURL := buildVisualizationURL(r, vizID, req.Boxes, packedBoxes)

	resp := PackResponse{
		PackedBoxes:      packedBoxes,
		UnpackedItems:    unpackedItems,
		TotalVolume:      totalBoxVolume,
		Utilization:      utilization,
		VisualizationURL: vizURL,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func buildVisualizationURL(r *http.Request, vizID string, boxes []InputBox, packedBoxes []PackedBox) string {
	vizData := VisualizationData{
		PackedBoxes: packedBoxes,
		Boxes:       boxes,
		RequestID:   vizID,
	}

	vizHTML, err := GenerateVisualizationHTML(vizData)
	if err != nil {
		return ""
	}

	visualizations.Store(vizID, vizHTML)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	return scheme + "://" + host + "/visualize/" + vizID
}

func handleVisualization(w http.ResponseWriter, r *http.Request) {
	vizID := strings.TrimPrefix(r.URL.Path, "/visualize/")
	if vizID == "" {
		http.Error(w, "Invalid visualization ID", http.StatusBadRequest)
		return
	}

	htmlContent, ok := visualizations.Load(vizID)
	if !ok {
		http.Error(w, "Visualization not found or expired", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(htmlContent.(string)))
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.FileServer(http.FS(fsys)).ServeHTTP(w, r)
}
