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
