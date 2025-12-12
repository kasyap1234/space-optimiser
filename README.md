# Space Optimiser (Bin Packer)

This is a Cloud Run service written in Go that performs 3D bin packing.

## Local Development

### Prerequisites
- Go 1.21+
- [pack](https://buildpacks.io/docs/tools/pack/) (optional, for container testing)

### Running Locally
You can run the service locally with the built-in HTTP server:

```bash
go run .
```

The function will be available at `http://localhost:8080`.

### Testing
Send a POST request to the `/pack` endpoint:

```bash
curl -X POST -H "Content-Type: application/json" -d @test_payload.json http://localhost:8080/pack
```

## API Response

The `/pack` endpoint returns:
- **packed_boxes**: List of boxes with packed items and their 3D coordinates
- **unpacked_items**: Items that couldn't fit in any box
- **total_volume**: Total volume of all boxes used
- **utilization_percent**: Percentage of box space utilized
- **visualization_data_uri**: Data URI for instant 3D visualization (paste into browser)
- **visualization_html**: Raw HTML string for saving and opening locally

### Viewing the Visualization

You can view the interactive 3D visualization in two ways:

1. **✅ Recommended - Save HTML File**: Copy the `visualization_html` from the response, save it as `visualization.html`, and open it in your browser. This method works reliably because local HTML files can load external JavaScript libraries.

   ```bash
   # Extract HTML from response and save
   curl ... | jq -r '.visualization_html' > visualization.html
   open visualization.html  # macOS
   ```

2. **⚠️ Limited - Data URI**: Copy the `visualization_data_uri` and paste it into your browser's address bar. **Note:** Due to browser security policies, the 3D visualization may not render in data URI contexts. If you don't see the 3D boxes, use method 1 instead.

## Deploying to Cloud Run

Build and deploy with Cloud Run (substitute your project/region/service names):

```bash
gcloud builds submit --tag gcr.io/PROJECT_ID/space-optimiser
gcloud run deploy space-optimiser \
  --image gcr.io/PROJECT_ID/space-optimiser \
  --region REGION \
  --allow-unauthenticated
```

Or, build locally with Cloud Buildpacks and run via Docker:

```bash
pack build space-optimiser --builder gcr.io/buildpacks/builder:v1
docker run --rm -p 8080:8080 space-optimiser
```

## 🌐 Live API

The API is live on RapidAPI marketplace and backed by Google Cloud Run:

**RapidAPI URL**: https://space-optimiser.p.rapidapi.com

**Endpoint**: `POST /pack`

Example request:
```bash
curl -X POST https://space-optimiser.p.rapidapi.com/pack \
  -H "Content-Type: application/json" \
  -H "X-RapidAPI-Key: YOUR_API_KEY" \
  -H "X-RapidAPI-Host: space-optimiser.p.rapidapi.com" \
  -d '{
    "items": [{"id": "item-1", "w": 10, "h": 10, "d": 10, "quantity": 1}],
    "boxes": [{"id": "box-1", "w": 30, "h": 30, "d": 30}]
  }'
```

## 📧 Contact

For questions, feedback, or issues:
- **Email**: kasyap3103@gmail.com
- **GitHub**: https://github.com/kasyap3103/space-optimiser
