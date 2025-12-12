package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
)

// VisualizationData contains all data needed to render the 3D visualization.
type VisualizationData struct {
	PackedBoxes []PackedBox
	Boxes       []InputBox
	RequestID   string
}

// GenerateVisualizationHTML creates an interactive 3D HTML visualization.
func GenerateVisualizationHTML(data VisualizationData) (string, error) {
	t, err := template.New("visualization").Funcs(template.FuncMap{
		"jsonMarshal": func(v any) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return "[]"
			}
			return template.JS(b)
		},
	}).Parse(visualizationTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

const visualizationTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>3D Packing Result - {{.RequestID}}</title>
    <style>
        :root {
            --bg-primary: #0f0f1a;
            --bg-secondary: #1a1a2e;
            --bg-tertiary: #252542;
            --text-primary: #e8e8f0;
            --text-secondary: #a0a0b8;
            --accent-primary: #6366f1;
            --accent-secondary: #818cf8;
            --success: #22c55e;
            --border-color: #3a3a5c;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Inter', 'Segoe UI', system-ui, sans-serif;
            background: var(--bg-primary);
            overflow: hidden;
            color: var(--text-primary);
        }
        #container { width: 100vw; height: 100vh; position: relative; }
        
        #info {
            position: absolute;
            top: 20px;
            left: 20px;
            background: var(--bg-secondary);
            padding: 20px;
            border-radius: 16px;
            box-shadow: 0 10px 40px rgba(0, 0, 0, 0.4);
            max-width: 280px;
            z-index: 100;
            border: 1px solid var(--border-color);
            backdrop-filter: blur(10px);
        }
        #info h2 {
            color: var(--accent-secondary);
            margin-bottom: 16px;
            font-size: 18px;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .stat {
            display: flex;
            justify-content: space-between;
            padding: 10px 0;
            border-bottom: 1px solid var(--border-color);
            font-size: 13px;
        }
        .stat:last-child { border-bottom: none; }
        .stat-label { color: var(--text-secondary); }
        .stat-value { font-weight: 600; color: var(--text-primary); }
        .stat-value.highlight { color: var(--success); }
        
        #controls {
            position: absolute;
            bottom: 20px;
            left: 20px;
            background: var(--bg-secondary);
            padding: 16px;
            border-radius: 12px;
            z-index: 100;
            border: 1px solid var(--border-color);
        }
        #controls h4 {
            font-size: 12px;
            color: var(--text-secondary);
            margin-bottom: 10px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        #controls p {
            margin: 6px 0;
            color: var(--text-secondary);
            font-size: 12px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .kbd {
            background: var(--bg-tertiary);
            padding: 3px 8px;
            border-radius: 4px;
            font-family: monospace;
            font-size: 11px;
            color: var(--text-primary);
        }
        
        .legend {
            position: absolute;
            top: 20px;
            right: 20px;
            background: var(--bg-secondary);
            padding: 16px;
            border-radius: 12px;
            z-index: 100;
            border: 1px solid var(--border-color);
            max-width: 220px;
        }
        .legend h3 {
            color: var(--accent-secondary);
            margin-bottom: 12px;
            font-size: 14px;
        }
        .legend-item {
            display: flex;
            align-items: center;
            margin: 8px 0;
            font-size: 12px;
        }
        .legend-color {
            width: 16px;
            height: 16px;
            border-radius: 4px;
            margin-right: 10px;
            border: 1px solid rgba(255,255,255,0.1);
        }
    </style>
</head>
<body>
    <div id="container"></div>
    
    <div id="info">
        <h2>📦 Packing Results</h2>
        <div class="stat">
            <span class="stat-label">Boxes Used</span>
            <span class="stat-value">{{len .PackedBoxes}}</span>
        </div>
        <div class="stat">
            <span class="stat-label">Total Items</span>
            <span class="stat-value highlight" id="totalItems">0</span>
        </div>
        <div class="stat">
            <span class="stat-label">Request ID</span>
            <span class="stat-value" style="font-size: 10px; word-break: break-all;">{{.RequestID}}</span>
        </div>
    </div>

    <div class="legend">
        <h3>🎨 Legend</h3>
        <div class="legend-item">
            <div class="legend-color" style="background: rgba(99, 102, 241, 0.7);"></div>
            <span>Box Container</span>
        </div>
        <div class="legend-item">
            <div class="legend-color" style="background: linear-gradient(135deg, #6366f1, #ec4899);"></div>
            <span>Packed Items</span>
        </div>
    </div>

    <div id="controls">
        <h4>🖱️ Controls</h4>
        <p><span class="kbd">Left Drag</span> Rotate</p>
        <p><span class="kbd">Right Drag</span> Pan</p>
        <p><span class="kbd">Scroll</span> Zoom</p>
    </div>

    <script src="https://cdnjs.cloudflare.com/ajax/libs/three.js/r128/three.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/three@0.128.0/examples/js/controls/OrbitControls.js"></script>
    
    <script>
        const scene = new THREE.Scene();
        scene.background = new THREE.Color(0x0f0f1a);
        scene.fog = new THREE.Fog(0x0f0f1a, 80, 300);
        
        const camera = new THREE.PerspectiveCamera(50, window.innerWidth / window.innerHeight, 0.1, 10000);
        
        const renderer = new THREE.WebGLRenderer({ antialias: true });
        renderer.setSize(window.innerWidth, window.innerHeight);
        renderer.shadowMap.enabled = true;
        renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        renderer.toneMapping = THREE.ACESFilmicToneMapping;
        document.getElementById('container').appendChild(renderer.domElement);
        
        // Lighting
        const ambientLight = new THREE.AmbientLight(0xffffff, 0.4);
        scene.add(ambientLight);
        
        const mainLight = new THREE.DirectionalLight(0xffffff, 1);
        mainLight.position.set(50, 100, 50);
        mainLight.castShadow = true;
        mainLight.shadow.mapSize.width = 2048;
        mainLight.shadow.mapSize.height = 2048;
        scene.add(mainLight);
        
        const fillLight = new THREE.DirectionalLight(0x6366f1, 0.3);
        fillLight.position.set(-50, 30, -50);
        scene.add(fillLight);
        
        const controls = new THREE.OrbitControls(camera, renderer.domElement);
        controls.enableDamping = true;
        controls.dampingFactor = 0.05;
        
        // Grid
        const gridHelper = new THREE.GridHelper(200, 40, 0x2a2a4a, 0x1a1a2e);
        scene.add(gridHelper);
        
        // Data
        const packedBoxes = {{.PackedBoxes | jsonMarshal}};
        const boxes = {{.Boxes | jsonMarshal}};
        
        let totalItems = 0;
        let maxDimension = 0;
        
        const boxMap = {};
        boxes.forEach(box => { boxMap[box.id] = box; });
        
        const colorPalette = [
            0x6366f1, 0xec4899, 0x14b8a6, 0xf59e0b, 
            0x8b5cf6, 0x06b6d4, 0xf43f5e, 0x22c55e
        ];
        
        packedBoxes.forEach((packedBox, boxIndex) => {
            const boxDef = boxMap[packedBox.box_id];
            if (!boxDef) return;
            
            maxDimension = Math.max(maxDimension, boxDef.w, boxDef.h, boxDef.d);
            
            const offsetX = boxIndex * (boxDef.w + 30);
            
            // Glass box
            const boxGeometry = new THREE.BoxGeometry(boxDef.w, boxDef.h, boxDef.d);
            const boxMaterial = new THREE.MeshPhysicalMaterial({
                color: 0xffffff,
                metalness: 0,
                roughness: 0,
                transmission: 0.9,
                transparent: true,
                opacity: 0.15,
                side: THREE.DoubleSide,
                depthWrite: false
            });
            const boxMesh = new THREE.Mesh(boxGeometry, boxMaterial);
            boxMesh.position.set(offsetX + boxDef.w / 2, boxDef.h / 2, boxDef.d / 2);
            scene.add(boxMesh);
            
            // Box edges
            const boxEdges = new THREE.EdgesGeometry(boxGeometry);
            const boxLine = new THREE.LineSegments(
                boxEdges,
                new THREE.LineBasicMaterial({ color: 0x6366f1, linewidth: 2 })
            );
            boxLine.position.copy(boxMesh.position);
            scene.add(boxLine);
            
            // Items
            packedBox.contents.forEach((item, itemIndex) => {
                totalItems++;
                
                const itemGeometry = new THREE.BoxGeometry(item.w * 0.98, item.h * 0.98, item.d * 0.98);
                const itemMaterial = new THREE.MeshStandardMaterial({
                    color: colorPalette[itemIndex % colorPalette.length],
                    roughness: 0.3,
                    metalness: 0.1
                });
                
                const itemMesh = new THREE.Mesh(itemGeometry, itemMaterial);
                itemMesh.position.set(
                    offsetX + item.x + item.w / 2,
                    item.y + item.h / 2,
                    item.z + item.d / 2
                );
                itemMesh.castShadow = true;
                itemMesh.receiveShadow = true;
                scene.add(itemMesh);
                
                // Item edges
                const itemEdges = new THREE.EdgesGeometry(itemGeometry);
                const itemLine = new THREE.LineSegments(
                    itemEdges,
                    new THREE.LineBasicMaterial({ color: 0x000000, opacity: 0.2, transparent: true })
                );
                itemLine.position.copy(itemMesh.position);
                scene.add(itemLine);
            });
        });
        
        document.getElementById('totalItems').textContent = totalItems;
        
        const cameraDistance = maxDimension * 2.5;
        camera.position.set(cameraDistance, cameraDistance * 0.8, cameraDistance);
        camera.lookAt(maxDimension / 2, 0, maxDimension / 2);
        
        function animate() {
            requestAnimationFrame(animate);
            controls.update();
            renderer.render(scene, camera);
        }
        animate();
        
        window.addEventListener('resize', () => {
            camera.aspect = window.innerWidth / window.innerHeight;
            camera.updateProjectionMatrix();
            renderer.setSize(window.innerWidth, window.innerHeight);
        });
    </script>
</body>
</html>`
