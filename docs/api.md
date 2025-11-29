# Space Optimizer API Documentation

## Overview

The Space Optimizer API provides a high-performance 3D bin packing service for optimizing item placement into containers/boxes. It uses an advanced Extreme Points algorithm with multi-strategy sorting (volume, longest dimension, surface area) and support for real-world constraints like weight limits, fragility, priorities, and adequate support (60% base support required).

**Key Features:**
- Multi-tier support for RapidAPI (Basic, Pro, Ultra, Mega) with usage limits
- 3D visualization generation with shareable temporary URLs
- Weight and fragility constraints
- Rotations and quantity support
- High utilization through multi-start optimization
- CORS-enabled for web integration

**Base URL:** `https://space-optimiser-rapidapi.p.rapidapi.com/` (or local: `http://localhost:8080`)

## Authentication & Tiers

The API uses RapidAPI subscription tiers via the `X-RapidAPI-Subscription` header. Optional `X-RapidAPI-Proxy-Secret` for proxy validation.

### Tier Limits
| Tier | Max Items | Max Boxes | Timeout | Viz TTL | Max Body Size | Visualization |
|------|-----------|-----------|---------|---------|---------------|---------------|
| [`basic`] | 50 | 10 | 10s | None | 1MB | ❌ |
| [`pro`] | 200 | 30 | 20s | 2min | 5MB | ✅ |
| [`ultra`] | 500 | 60 | 30s | 5min | 10MB | ✅ |
| [`mega`] | 1000 | 100 | 30s | 10min | 10MB | ✅ |

**Header Example:**
```
X-RapidAPI-Subscription: pro
```

## Data Schemas

### [`PackRequest`](main.go:230)
```json
{
  "items": [InputItem],
  "boxes": [InputBox]
}
```

#### [`InputItem`](packer.go:11)
```json
{
  "id": "string",          // Unique identifier
  "w": 1-10000,            // Width (integer)
  "h": 1-10000,            // Height (integer)
  "d": 1-10000,            // Depth (integer)
  "quantity": 1-1000,      // Number of identical items
  "weight": 0.0+,          // Optional: kg (default 0)
  "fragile": false,        // Optional: Cannot have items above
  "priority": 0            // Optional: Higher = packed first
}
```

#### [`InputBox`](packer.go:22)
```json
{
  "id": "string",          // Unique identifier
  "w": 1-10000,            // Width
  "h": 1-10000,            // Height
  "d": 1-10000,            // Depth
  "max_weight": 0.0+       // Optional: kg capacity (default unlimited)
}
```

### [`PackResponse`](main.go:235)
```json
{
  "packed_boxes": [PackedBox],
  "unpacked_items": [InputItem],  // Items that couldn't fit
  "total_volume": 0,              // Total volume of used boxes
  "utilization_percent": 0.0      // % utilization (items vol / boxes vol)
}
```

#### [`PackedBox`](packer.go:30)
```json
{
  "box_id": "string",
  "contents": [Placement]
}
```

#### [`Placement`](packer.go:35)
```json
{
  "item_id": "string",
  "x": 0, "y": 0, "z": 0,    // Position (bottom-front-left origin)
  "w": int, "h": int, "d": int,  // Rotated dimensions
  "weight": 0.0,             // Item weight
  "fragile": false
}
```

### [`VisualizeResponse`](main.go:242)
```json
{
  "url": "/view/{unique_id}",
  "expires_in_seconds": 300
}
```

## Endpoints

### `POST /pack`
**Pack items into boxes (core endpoint)**

**Request:** [`PackRequest`](#packrequest) (JSON, tier-based size limit)

**Response:** 200 [`PackResponse`](#packresponse)

**Errors:**
- `400 Bad Request`: Invalid JSON, missing items/boxes, validation failed (e.g., dims ≤0, too many items)
- `413 Payload Too Large`: Exceeds tier body limit
- `408 Request Timeout`: Packing exceeded tier timeout

**Example:**
```bash
curl -X POST "http://localhost:8080/pack" \
  -H "Content-Type: application/json" \
  -H "X-RapidAPI-Subscription: pro" \
  -d @test_payload.json
```

### `POST /visualize`
**Pack + generate shareable 3D visualization** (Pro+ tiers only)

**Request:** [`PackRequest`](#packrequest)

**Response:** 200 [`VisualizeResponse`](#visualizeresponse)

**Errors:** Same as `/pack` + `403 Forbidden` (tier lacks viz support)

**Usage:** Visit `url` for interactive 3D view (Three.js, OrbitControls). Expires per tier.

### `GET /view/{id}`
**Serve visualization page** (public, expires automatically)

**Response:** 200 HTML (embedded JSON data + Three.js viewer)

**Errors:**
- `400 Bad Request`: Invalid/missing ID
- `404 Not Found`: Expired or invalid ID

### `GET /healthz`
**Health check**

**Response:** `{"status":"ok"}`

### Static Assets
- `GET /` & `GET /*` → Serve `static/` files (index.html, JS/CSS)

## Examples

### Test Payload ([`test_payload.json`](test_payload.json:1))
```json
{
  "items": [
    {"id": "item-b", "w": 20, "h": 20, "d": 20, "quantity": 1},
    {"id": "item-a", "w": 10, "h": 10, "d": 10, "quantity": 2},
    {"id": "item-c", "w": 5, "h": 5, "d": 5, "quantity": 5}
  ],
  "boxes": [
    {"id": "box-small", "w": 15, "h": 15, "d": 15},
    {"id": "box-large", "w": 30, "h": 30, "d": 30}
  ]
}
```

**Expected:** All items packed into `box-large` (10625/27000 vol ~39%).

## Algorithm Details
- **Extreme Points (EP)**: Generates candidate positions from item corners
- **Multi-Strategy**: Tries volume/largest-dim/surface-area sorting, picks best
- **DBLF Scoring**: Deepest-Bottom-Left-Fill + wall/corner bonuses
- **Constraints**: 60% support, no stacking on fragile, weight limits, rotations
- **Fallback**: Boxes tried smallest-first, best volume fit selected

## Error Codes
| Code | Message | Cause |
|------|---------|-------|
| 400 | "Invalid JSON payload" | Malformed request |
| 400 | "Items and boxes are required" | Empty arrays |
| 400 | "Validation error: ..." | Dim/quantity limits |
| 413 | "Request body too large" | Tier limit exceeded |
| 408 | "Packing operation timed out" | Tier timeout |
| 403 | "Visualization not available..." | Tier restriction |
| 404 | "Visualization not found..." | Expired ID |
| 405 | "Method not allowed" | Wrong HTTP method |

## Environment Variables (Deployment)
- `PORT`: Server port (default 8080)
- `ALLOWED_ORIGINS`: CORS origins (default "*")
- `RAPIDAPI_PROXY_SECRET`: Proxy validation (optional)

## Deployment Modes
- **Standalone**: `go run main.go`
- **Google Cloud Functions**: Auto-detects via `FUNCTION_TARGET`
- **Docker**: Supports embedded static assets

## Rate Limits & Best Practices
- Respect tier limits to avoid 413/408
- Use `/visualize` for client-side 3D previews
- Pre-validate inputs client-side
- Items with `priority > 0` packed first

**OpenAPI Spec:** Auto-generated docs available at `/docs` (future).

---
*Generated from source code analysis. Last updated: 2025-11-29*