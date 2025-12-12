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

1. **Quick Preview**: Copy the `visualization_data_uri` from the response and paste it into your browser's address bar
2. **Save File**: Copy the `visualization_html`, save it as `visualization.html`, and open it locally

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
