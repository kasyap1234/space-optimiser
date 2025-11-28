package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	funcframework "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

//go:embed static/*
var staticFiles embed.FS

// Configuration
const (
	MaxRequestBodySize = 10 * 1024 * 1024 // 10 MB
	PackTimeout        = 30 * time.Second // 30 seconds
	staticCacheControl = "public, max-age=600"
	ViewCacheTTL       = 5 * time.Minute // Visualization cache TTL
	CleanupInterval    = 1 * time.Minute // How often to run cleanup
)

var (
	AllowedOrigins = getEnv("ALLOWED_ORIGINS", "*") // Configurable CORS
	handlerOnce    sync.Once
	appHandler     http.Handler
)

// --- Visualization Cache ---

// CachedVisualization holds packing result for temporary viewing
type CachedVisualization struct {
	Request   PackRequest  `json:"request"`
	Response  PackResponse `json:"response"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// VisualizationCache is a thread-safe in-memory cache with TTL
type VisualizationCache struct {
	mu    sync.RWMutex
	items map[string]*CachedVisualization
}

var vizCache = &VisualizationCache{
	items: make(map[string]*CachedVisualization),
}

// Set stores a visualization with TTL
func (c *VisualizationCache) Set(id string, viz *CachedVisualization) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[id] = viz
}

// Get retrieves a visualization if it exists and hasn't expired
func (c *VisualizationCache) Get(id string) (*CachedVisualization, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	viz, exists := c.items[id]
	if !exists {
		return nil, false
	}

	if time.Now().After(viz.ExpiresAt) {
		return nil, false
	}

	return viz, true
}

// Cleanup removes expired entries
func (c *VisualizationCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for id, viz := range c.items {
		if now.After(viz.ExpiresAt) {
			delete(c.items, id)
		}
	}
}

// startCleanupRoutine periodically cleans up expired cache entries
func startCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(CleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			vizCache.Cleanup()
		}
	}()
}

// generateID creates a random URL-safe ID
func generateID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// --- Request/Response Types ---

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

type VisualizeResponse struct {
	URL              string `json:"url"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

// --- HTTP Handlers ---

func getAppHandler() http.Handler {
	handlerOnce.Do(func() {
		staticHandler, err := newStaticHandler()
		if err != nil {
			log.Fatalf("failed to initialise static file handler: %v", err)
		}

		// Start cleanup routine for visualization cache
		startCleanupRoutine()

		mux := http.NewServeMux()
		mux.Handle("/healthz", http.HandlerFunc(handleHealth))
		mux.Handle("/pack", http.HandlerFunc(handlePack))
		mux.Handle("/visualize", http.HandlerFunc(handleVisualize))
		mux.Handle("/view/", http.HandlerFunc(handleView))
		mux.Handle("/", staticHandler)

		appHandler = withMiddleware(mux)
	})

	return appHandler
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", AllowedOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func newStaticHandler() (http.Handler, error) {
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("Cache-Control") == "" {
			w.Header().Set("Cache-Control", staticCacheControl)
		}
		fileServer.ServeHTTP(w, r)
	}), nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// Packer is the entry point for Google Cloud Functions
func Packer(w http.ResponseWriter, r *http.Request) {
	getAppHandler().ServeHTTP(w, r)
}

func handlePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req, resp, err := processPackRequest(r)
	if err != nil {
		handlePackError(w, err)
		return
	}
	_ = req // Not needed for pack response

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(resp); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// handleVisualize processes packing and stores result for visualization
func handleVisualize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req, resp, err := processPackRequest(r)
	if err != nil {
		handlePackError(w, err)
		return
	}

	// Generate unique ID and store in cache
	id := generateID()
	now := time.Now()
	cached := &CachedVisualization{
		Request:   *req,
		Response:  *resp,
		CreatedAt: now,
		ExpiresAt: now.Add(ViewCacheTTL),
	}
	vizCache.Set(id, cached)

	// Return the URL
	vizResp := VisualizeResponse{
		URL:              "/view/" + id,
		ExpiresInSeconds: int(ViewCacheTTL.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(vizResp); err != nil {
		log.Printf("failed to encode visualize response: %v", err)
	}
}

// handleView serves the standalone visualization HTML
func handleView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path: /view/{id}
	path := strings.TrimPrefix(r.URL.Path, "/view/")
	id := strings.TrimSuffix(path, "/")

	if id == "" {
		http.Error(w, "Missing visualization ID", http.StatusBadRequest)
		return
	}

	viz, found := vizCache.Get(id)
	if !found {
		http.Error(w, "Visualization not found or expired", http.StatusNotFound)
		return
	}

	// Render the standalone viewer HTML with embedded data
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	if err := renderViewer(w, viz); err != nil {
		log.Printf("failed to render viewer: %v", err)
		http.Error(w, "Failed to render visualization", http.StatusInternalServerError)
	}
}

// processPackRequest handles common request parsing and packing logic
func processPackRequest(r *http.Request) (*PackRequest, *PackResponse, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, MaxRequestBodySize)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req PackRequest
	if err := decoder.Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			return nil, nil, &packError{code: http.StatusRequestEntityTooLarge, msg: "Request body too large"}
		}
		return nil, nil, &packError{code: http.StatusBadRequest, msg: "Invalid JSON payload"}
	}

	if len(req.Items) == 0 || len(req.Boxes) == 0 {
		return nil, nil, &packError{code: http.StatusBadRequest, msg: "Items and boxes are required"}
	}

	if err := ValidateInputs(req.Items, req.Boxes); err != nil {
		return nil, nil, &packError{code: http.StatusBadRequest, msg: fmt.Sprintf("Validation error: %v", err)}
	}

	ctx, cancel := context.WithTimeout(context.Background(), PackTimeout)
	defer cancel()

	type packResult struct {
		packedBoxes   []PackedBox
		unpackedItems []InputItem
	}

	resultChan := make(chan packResult, 1)
	go func() {
		packed, unpacked := Pack(req.Items, req.Boxes)
		select {
		case resultChan <- packResult{packed, unpacked}:
		case <-ctx.Done():
		}
	}()

	var (
		packedBoxes   []PackedBox
		unpackedItems []InputItem
	)

	select {
	case result := <-resultChan:
		packedBoxes = result.packedBoxes
		unpackedItems = result.unpackedItems
	case <-ctx.Done():
		return nil, nil, &packError{code: http.StatusRequestTimeout, msg: "Packing operation timed out"}
	}

	boxMap := make(map[string]InputBox)
	for _, box := range req.Boxes {
		boxMap[box.ID] = box
	}

	totalBoxVolume := 0
	totalItemVolume := 0

	for _, box := range packedBoxes {
		if b, ok := boxMap[box.BoxID]; ok {
			totalBoxVolume += b.W * b.H * b.D
		}
		for _, item := range box.Contents {
			totalItemVolume += item.W * item.H * item.D
		}
	}

	utilization := 0.0
	if totalBoxVolume > 0 {
		utilization = (float64(totalItemVolume) / float64(totalBoxVolume)) * 100
	}

	resp := &PackResponse{
		PackedBoxes:   packedBoxes,
		UnpackedItems: unpackedItems,
		TotalVolume:   totalBoxVolume,
		Utilization:   utilization,
	}

	return &req, resp, nil
}

type packError struct {
	code int
	msg  string
}

func (e *packError) Error() string {
	return e.msg
}

func handlePackError(w http.ResponseWriter, err error) {
	if pe, ok := err.(*packError); ok {
		http.Error(w, pe.msg, pe.code)
		return
	}
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

// renderViewer generates standalone HTML with embedded visualization data
func renderViewer(w http.ResponseWriter, viz *CachedVisualization) error {
	// Serialize request and response to JSON for embedding
	requestJSON, err := json.Marshal(viz.Request)
	if err != nil {
		return err
	}
	responseJSON, err := json.Marshal(viz.Response)
	if err != nil {
		return err
	}

	data := struct {
		RequestJSON  template.JS
		ResponseJSON template.JS
		Utilization  float64
		TotalVolume  int
		UnpackedCount int
		BoxCount     int
	}{
		RequestJSON:  template.JS(requestJSON),
		ResponseJSON: template.JS(responseJSON),
		Utilization:  viz.Response.Utilization,
		TotalVolume:  viz.Response.TotalVolume,
		UnpackedCount: len(viz.Response.UnpackedItems),
		BoxCount:     len(viz.Response.PackedBoxes),
	}

	return viewerTemplate.Execute(w, data)
}

// Viewer template - standalone HTML with Three.js visualization
var viewerTemplate = template.Must(template.New("viewer").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>3D Packing Visualization</title>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@300;400;600&display=swap');
        
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: 'JetBrains Mono', monospace;
            background: linear-gradient(135deg, #0f0f1a 0%, #1a1a2e 50%, #16213e 100%);
            color: #e0e0e0;
            height: 100vh;
            overflow: hidden;
        }
        
        #canvas-container {
            width: 100vw;
            height: 100vh;
            position: relative;
        }
        
        #stats-overlay {
            position: absolute;
            top: 20px;
            left: 20px;
            background: rgba(15, 15, 26, 0.9);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(99, 102, 241, 0.3);
            border-radius: 12px;
            padding: 20px 25px;
            z-index: 100;
            min-width: 220px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
        }
        
        #stats-overlay h2 {
            color: #818cf8;
            font-size: 14px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 2px;
            margin-bottom: 15px;
            padding-bottom: 10px;
            border-bottom: 1px solid rgba(99, 102, 241, 0.2);
        }
        
        .stat-row {
            display: flex;
            justify-content: space-between;
            margin-bottom: 10px;
            font-size: 13px;
        }
        
        .stat-label {
            color: #94a3b8;
        }
        
        .stat-value {
            color: #f8fafc;
            font-weight: 600;
        }
        
        .stat-value.highlight {
            color: #34d399;
        }
        
        .stat-value.warning {
            color: #fbbf24;
        }
        
        #box-selector {
            position: absolute;
            bottom: 20px;
            left: 50%;
            transform: translateX(-50%);
            background: rgba(15, 15, 26, 0.9);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(99, 102, 241, 0.3);
            border-radius: 8px;
            padding: 12px 20px;
            z-index: 100;
            display: flex;
            align-items: center;
            gap: 15px;
        }
        
        #box-selector label {
            color: #94a3b8;
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        
        #box-selector select {
            background: rgba(30, 30, 50, 0.8);
            border: 1px solid rgba(99, 102, 241, 0.4);
            color: #f8fafc;
            padding: 8px 15px;
            border-radius: 6px;
            font-family: inherit;
            font-size: 13px;
            cursor: pointer;
            outline: none;
            transition: border-color 0.2s;
        }
        
        #box-selector select:hover {
            border-color: #818cf8;
        }
        
        #tooltip {
            position: absolute;
            background: rgba(15, 15, 26, 0.95);
            backdrop-filter: blur(8px);
            border: 1px solid rgba(99, 102, 241, 0.4);
            color: #f8fafc;
            padding: 12px 16px;
            border-radius: 8px;
            font-size: 12px;
            pointer-events: none;
            display: none;
            z-index: 200;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
            line-height: 1.6;
        }
        
        #tooltip strong {
            color: #818cf8;
        }

        #controls-hint {
            position: absolute;
            bottom: 80px;
            right: 20px;
            background: rgba(15, 15, 26, 0.7);
            border: 1px solid rgba(99, 102, 241, 0.2);
            border-radius: 8px;
            padding: 12px 16px;
            font-size: 11px;
            color: #64748b;
            z-index: 100;
        }
        
        #controls-hint div {
            margin-bottom: 4px;
        }
        
        #controls-hint span {
            color: #94a3b8;
        }
    </style>
    <script type="importmap">
        {
            "imports": {
                "three": "https://unpkg.com/three@0.160.0/build/three.module.js",
                "three/addons/": "https://unpkg.com/three@0.160.0/examples/jsm/"
            }
        }
    </script>
</head>
<body>
    <div id="canvas-container">
        <div id="stats-overlay">
            <h2>Packing Results</h2>
            <div class="stat-row">
                <span class="stat-label">Boxes Used</span>
                <span class="stat-value">{{.BoxCount}}</span>
            </div>
            <div class="stat-row">
                <span class="stat-label">Total Volume</span>
                <span class="stat-value">{{.TotalVolume}}</span>
            </div>
            <div class="stat-row">
                <span class="stat-label">Utilization</span>
                <span class="stat-value highlight">{{printf "%.1f" .Utilization}}%</span>
            </div>
            <div class="stat-row">
                <span class="stat-label">Unpacked</span>
                <span class="stat-value {{if gt .UnpackedCount 0}}warning{{end}}">{{.UnpackedCount}}</span>
            </div>
        </div>
        
        <div id="box-selector" style="display: none;">
            <label for="boxSelect">View Box:</label>
            <select id="boxSelect"></select>
        </div>
        
        <div id="controls-hint">
            <div><span>Rotate:</span> Left Mouse</div>
            <div><span>Pan:</span> Right Mouse</div>
            <div><span>Zoom:</span> Scroll</div>
        </div>
        
        <div id="tooltip"></div>
    </div>

    <script type="module">
        import * as THREE from 'three';
        import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

        // Embedded data from server
        const requestData = {{.RequestJSON}};
        const responseData = {{.ResponseJSON}};

        let scene, camera, renderer, controls;
        let currentMeshes = [];
        let raycaster, mouse;
        let hoveredItem = null;

        function init() {
            const container = document.getElementById('canvas-container');
            scene = new THREE.Scene();
            scene.background = new THREE.Color(0x0f0f1a);
            scene.fog = new THREE.Fog(0x0f0f1a, 80, 250);

            camera = new THREE.PerspectiveCamera(45, container.clientWidth / container.clientHeight, 0.1, 1000);
            camera.position.set(60, 45, 60);
            camera.lookAt(0, 0, 0);

            renderer = new THREE.WebGLRenderer({ antialias: true });
            renderer.setSize(container.clientWidth, container.clientHeight);
            renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
            renderer.shadowMap.enabled = true;
            renderer.shadowMap.type = THREE.PCFSoftShadowMap;
            renderer.toneMapping = THREE.ACESFilmicToneMapping;
            renderer.toneMappingExposure = 1.2;
            container.appendChild(renderer.domElement);

            controls = new OrbitControls(camera, renderer.domElement);
            controls.enableDamping = true;
            controls.dampingFactor = 0.05;
            controls.minDistance = 10;
            controls.maxDistance = 300;

            // Lighting
            const ambientLight = new THREE.AmbientLight(0x404060, 0.5);
            scene.add(ambientLight);

            const dirLight = new THREE.DirectionalLight(0xffffff, 1.2);
            dirLight.position.set(50, 100, 50);
            dirLight.castShadow = true;
            dirLight.shadow.mapSize.width = 2048;
            dirLight.shadow.mapSize.height = 2048;
            dirLight.shadow.camera.near = 10;
            dirLight.shadow.camera.far = 300;
            dirLight.shadow.camera.left = -100;
            dirLight.shadow.camera.right = 100;
            dirLight.shadow.camera.top = 100;
            dirLight.shadow.camera.bottom = -100;
            scene.add(dirLight);

            const fillLight = new THREE.DirectionalLight(0x6366f1, 0.4);
            fillLight.position.set(-50, 30, -50);
            scene.add(fillLight);

            const rimLight = new THREE.DirectionalLight(0x22d3ee, 0.3);
            rimLight.position.set(0, -50, 50);
            scene.add(rimLight);

            // Floor
            const planeGeometry = new THREE.PlaneGeometry(400, 400);
            const planeMaterial = new THREE.MeshStandardMaterial({ 
                color: 0x1a1a2e,
                roughness: 0.9,
                metalness: 0.1
            });
            const plane = new THREE.Mesh(planeGeometry, planeMaterial);
            plane.rotation.x = -Math.PI / 2;
            plane.position.y = -0.5;
            plane.receiveShadow = true;
            scene.add(plane);

            // Grid
            const gridHelper = new THREE.GridHelper(200, 40, 0x4338ca, 0x1e1b4b);
            gridHelper.position.y = -0.4;
            scene.add(gridHelper);

            // Raycaster for hover
            raycaster = new THREE.Raycaster();
            mouse = new THREE.Vector2();

            window.addEventListener('resize', onWindowResize);
            container.addEventListener('mousemove', onMouseMove);

            // Setup box selector
            setupBoxSelector();
            
            animate();
        }

        function setupBoxSelector() {
            const boxSelect = document.getElementById('boxSelect');
            const selector = document.getElementById('box-selector');
            
            if (responseData.packed_boxes && responseData.packed_boxes.length > 0) {
                selector.style.display = 'flex';
                
                responseData.packed_boxes.forEach((box, index) => {
                    const option = document.createElement('option');
                    option.value = index;
                    option.text = box.box_id + ' (#' + (index + 1) + ')';
                    boxSelect.appendChild(option);
                });

                renderBox(0);

                boxSelect.addEventListener('change', (e) => {
                    renderBox(parseInt(e.target.value));
                });
            }
        }

        function onWindowResize() {
            const container = document.getElementById('canvas-container');
            camera.aspect = container.clientWidth / container.clientHeight;
            camera.updateProjectionMatrix();
            renderer.setSize(container.clientWidth, container.clientHeight);
        }

        function onMouseMove(event) {
            const container = document.getElementById('canvas-container');
            const rect = container.getBoundingClientRect();
            mouse.x = ((event.clientX - rect.left) / container.clientWidth) * 2 - 1;
            mouse.y = -((event.clientY - rect.top) / container.clientHeight) * 2 + 1;

            const tooltip = document.getElementById('tooltip');
            if (hoveredItem) {
                tooltip.style.left = (event.clientX + 15) + 'px';
                tooltip.style.top = (event.clientY + 15) + 'px';
            }
        }

        function animate() {
            requestAnimationFrame(animate);
            controls.update();

            raycaster.setFromCamera(mouse, camera);
            const intersects = raycaster.intersectObjects(currentMeshes);

            const tooltip = document.getElementById('tooltip');

            if (intersects.length > 0) {
                const hit = intersects.find(i => i.object.userData && i.object.userData.isItem);

                if (hit) {
                    const item = hit.object.userData;
                    if (hoveredItem !== hit.object) {
                        if (hoveredItem) hoveredItem.material.emissive.setHex(0x000000);

                        hoveredItem = hit.object;
                        hoveredItem.material.emissive.setHex(0x222244);

                        tooltip.style.display = 'block';
                        tooltip.innerHTML = 
                            '<strong>ID:</strong> ' + item.id + '<br>' +
                            '<strong>Size:</strong> ' + item.w + ' × ' + item.h + ' × ' + item.d + '<br>' +
                            '<strong>Position:</strong> (' + item.x + ', ' + item.y + ', ' + item.z + ')';
                    }
                } else {
                    if (hoveredItem) {
                        hoveredItem.material.emissive.setHex(0x000000);
                        hoveredItem = null;
                        tooltip.style.display = 'none';
                    }
                }
            } else {
                if (hoveredItem) {
                    hoveredItem.material.emissive.setHex(0x000000);
                    hoveredItem = null;
                    tooltip.style.display = 'none';
                }
            }

            renderer.render(scene, camera);
        }

        function clearScene() {
            currentMeshes.forEach(mesh => {
                scene.remove(mesh);
                if (mesh.geometry) mesh.geometry.dispose();
                if (mesh.material) {
                    if (Array.isArray(mesh.material)) {
                        mesh.material.forEach(m => m.dispose());
                    } else {
                        mesh.material.dispose();
                    }
                }
            });
            currentMeshes = [];
        }

        function getItemColor(id) {
            // Generate vibrant colors from item ID
            let hash = 0;
            for (let i = 0; i < id.length; i++) {
                hash = id.charCodeAt(i) + ((hash << 5) - hash);
            }
            
            // Use HSL for more vibrant colors
            const hue = Math.abs(hash % 360);
            const saturation = 65 + (hash % 20);
            const lightness = 50 + (hash % 15);
            
            return new THREE.Color('hsl(' + hue + ', ' + saturation + '%, ' + lightness + '%)');
        }

        function drawContainer(w, h, d) {
            // Glass container
            const geometry = new THREE.BoxGeometry(w, h, d);
            const material = new THREE.MeshPhysicalMaterial({
                color: 0x6366f1,
                metalness: 0.1,
                roughness: 0.1,
                transmission: 0.85,
                transparent: true,
                opacity: 0.15,
                side: THREE.DoubleSide,
                depthWrite: false
            });

            const boxMesh = new THREE.Mesh(geometry, material);
            boxMesh.position.set(w / 2, h / 2, d / 2);
            boxMesh.userData = { isBox: true };
            scene.add(boxMesh);
            currentMeshes.push(boxMesh);

            // Glowing edges
            const edges = new THREE.EdgesGeometry(geometry);
            const lineMaterial = new THREE.LineBasicMaterial({ 
                color: 0x818cf8,
                transparent: true,
                opacity: 0.6
            });
            const line = new THREE.LineSegments(edges, lineMaterial);
            line.position.copy(boxMesh.position);
            scene.add(line);
            currentMeshes.push(line);
        }

        function drawItem(item) {
            const geometry = new THREE.BoxGeometry(item.w * 0.98, item.h * 0.98, item.d * 0.98);
            const color = getItemColor(item.item_id);

            const material = new THREE.MeshStandardMaterial({
                color: color,
                roughness: 0.4,
                metalness: 0.3,
            });

            const cube = new THREE.Mesh(geometry, material);
            cube.position.set(
                item.x + item.w / 2,
                item.y + item.h / 2,
                item.z + item.d / 2
            );
            cube.castShadow = true;
            cube.receiveShadow = true;

            cube.userData = {
                isItem: true,
                id: item.item_id,
                w: item.w, h: item.h, d: item.d,
                x: item.x, y: item.y, z: item.z
            };

            // Subtle edges
            const edges = new THREE.EdgesGeometry(geometry);
            const line = new THREE.LineSegments(edges, new THREE.LineBasicMaterial({ 
                color: 0x000000, 
                opacity: 0.2, 
                transparent: true 
            }));
            line.position.copy(cube.position);

            scene.add(cube);
            scene.add(line);
            currentMeshes.push(cube);
            currentMeshes.push(line);
        }

        function renderBox(index) {
            clearScene();

            const packedBox = responseData.packed_boxes[index];
            const boxDef = requestData.boxes.find(b => b.id === packedBox.box_id);
            
            if (boxDef) {
                drawContainer(boxDef.w, boxDef.h, boxDef.d);

                // Adjust camera
                const maxDim = Math.max(boxDef.w, boxDef.h, boxDef.d);
                camera.position.set(maxDim * 1.8, maxDim * 1.2, maxDim * 1.8);
                controls.target.set(boxDef.w / 2, boxDef.h / 2, boxDef.d / 2);
                controls.update();
            }

            if (packedBox.contents) {
                packedBox.contents.forEach(item => drawItem(item));
            }
        }

        init();
    </script>
</body>
</html>`))

func main() {
	if shouldUseFunctionsFramework() {
		if err := startFunctionsFramework(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := startStandaloneServer(); err != nil {
		log.Fatal(err)
	}
}

func shouldUseFunctionsFramework() bool {
	if os.Getenv("FUNCTION_TARGET") != "" {
		return true
	}
	if os.Getenv("GOOGLE_FUNCTION_TARGET") != "" {
		return true
	}
	return false
}

func startFunctionsFramework() error {
	ctx := context.Background()
	port := ensurePortEnv()
	log.Printf("Starting Functions Framework on port %s", port)
	if err := funcframework.RegisterHTTPFunctionContext(ctx, "/", Packer); err != nil {
		return fmt.Errorf("register http function: %w", err)
	}
	return funcframework.Start(port)
}

func startStandaloneServer() error {
	port := ensurePortEnv()
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           getAppHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       45 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("Server starting on port %s (standalone)", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func ensurePortEnv() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		if err := os.Setenv("PORT", port); err != nil {
			log.Printf("warning: unable to set PORT env: %v", err)
		}
	}
	return port
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
