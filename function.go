package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/google/uuid"
)

//go:embed static/*
var staticFiles embed.FS

// PackRequest defines the input structure for the packing API.
type PackRequest struct {
	Items []InputItem `json:"items"`
	Boxes []InputBox  `json:"boxes"`
}

// PackResponse defines the output structure for the packing API.
type PackResponse struct {
	PackedBoxes          []PackedBox `json:"packed_boxes"`
	UnpackedItems        []InputItem `json:"unpacked_items"`
	TotalVolume          int         `json:"total_volume"`
	Utilization          float64     `json:"utilization_percent"`
	VisualizationDataURI string      `json:"visualization_data_uri"`
	VisualizationHTML    string      `json:"visualization_html"`
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

	// Generate visualization HTML
	vizID := uuid.New().String()
	vizData := VisualizationData{
		PackedBoxes: packedBoxes,
		Boxes:       req.Boxes,
		RequestID:   vizID,
	}

	vizHTML, err := GenerateVisualizationHTML(vizData)
	if err != nil {
		http.Error(w, "Failed to generate visualization", http.StatusInternalServerError)
		return
	}

	// Create data URI (base64 encoded)
	vizDataURI := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(vizHTML))

	resp := PackResponse{
		PackedBoxes:          packedBoxes,
		UnpackedItems:        unpackedItems,
		TotalVolume:          totalBoxVolume,
		Utilization:          utilization,
		VisualizationDataURI: vizDataURI,
		VisualizationHTML:    vizHTML,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.FileServer(http.FS(fsys)).ServeHTTP(w, r)
}
