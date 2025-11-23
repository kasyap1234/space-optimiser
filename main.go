package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static/*
var staticFiles embed.FS

type PackRequest struct {
	Items []InputItem `json:"items"`
	Boxes []InputBox  `json:"boxes"`
}

type PackResponse struct {
	PackedBoxes   []PackedBox `json:"packed_boxes"`
	UnpackedItems []InputItem `json:"unpacked_items"`
	TotalVolume   int         `json:"total_volume"`
	Utilization   float64     `json:"utilization_percent"`
}

// Packer is the entry point for Google Cloud Functions
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

	// Static file handling
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

	resp := PackResponse{
		PackedBoxes:   packedBoxes,
		UnpackedItems: unpackedItems,
		TotalVolume:   totalBoxVolume,
		Utilization:   utilization,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	// Create a sub-filesystem for the "static" directory
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Serve files
	http.FileServer(http.FS(fsys)).ServeHTTP(w, r)
}

func main() {
	// Local development server
	http.HandleFunc("/", Packer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
