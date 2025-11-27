# Deployment Guide

This guide covers deploying Fish-Speech-Go in various environments.

## 🐳 Docker Compose (Recommended)

The simplest deployment method for single-server setups.

### Requirements

- Docker 20.10+
- Docker Compose 2.0+
- NVIDIA GPU with CUDA 12.1+
- NVIDIA Container Toolkit
- 8GB+ GPU VRAM
- 16GB+ System RAM

### Steps

```bash
# 1. Clone repository
git clone https://github.com/fish-speech-go/fish-speech-go.git
cd fish-speech-go/docker

# 2. Configure environment
cp .env.example .env
nano .env  # Add your HF_TOKEN

# 3. Start services
docker compose up -d

# 4. Verify
docker compose ps
curl http://localhost:8080/v1/health
```

### Production Considerations

```yaml
# docker-compose.prod.yml
services:
  server:
    restart: always
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    deploy:
      resources:
        limits:
          memory: 1G

  inference:
    restart: always
    logging:
      driver: "json-file"
      options:
        max-size: "50m"
        max-file: "5"
```

## ☸️ Kubernetes

For scalable, production deployments.

### Helm Chart (Coming Soon)

```bash
helm repo add fish-speech-go https://...
helm install fish-speech fish-speech-go/fish-speech-go \
  --set hfToken=hf_xxx \
  --set gpu.enabled=true
```

### Manual Deployment

See `docs/kubernetes/` for manifests.

## 🔒 Security

### API Authentication

Set `API_KEY` in your environment:

```env
API_KEY=your-secure-api-key
```

Clients must include the header:
```bash
curl -H "Authorization: Bearer your-secure-api-key" ...
```

### Network Security

- Run behind a reverse proxy (nginx, traefik)
- Use HTTPS in production
- Restrict network access to inference container

### Example Nginx Config

```nginx
server {
    listen 443 ssl;
    server_name tts.yourdomain.com;

    ssl_certificate /etc/ssl/certs/your-cert.pem;
    ssl_certificate_key /etc/ssl/private/your-key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 📊 Monitoring

### Health Checks

```bash
# Liveness
curl http://localhost:8080/v1/health

# For load balancers
GET /v1/health -> 200 OK = healthy
```

### Logs

```bash
# View logs
docker compose logs -f

# Server logs only
docker compose logs -f server

# Inference logs only  
docker compose logs -f inference
```

### Metrics (Coming Soon)

Prometheus metrics endpoint at `/metrics`.

## 🔄 Updates

```bash
cd fish-speech-go
git pull
cd docker
docker compose down
docker compose build --no-cache
docker compose up -d
```

## 🆘 Troubleshooting

### Container won't start

```bash
# Check logs
docker compose logs inference

# Verify GPU access
docker run --rm --gpus all nvidia/cuda:12.1.0-base-ubuntu22.04 nvidia-smi
```

### Out of memory

- Reduce batch size
- Use smaller model variant
- Add swap space

### Slow inference

- Check GPU utilization: `nvidia-smi`
- Ensure CUDA is being used (not CPU)
- Check for thermal throttling
