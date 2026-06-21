# Demo

Start the stack:

```powershell
docker compose -f examples/docker-compose.yml up --build
```

Open the UI:

```text
http://localhost:8080
```

Search examples:

```text
q=failed
label=service=shop
level=info
since=1h
```

Send a single HTTP event:

```powershell
curl.exe -X POST http://localhost:8080/api/logs `
  -H "content-type: application/json" `
  -d "{\"level\":\"error\",\"message\":\"payment failed password=hunter2\",\"labels\":{\"service\":\"billing\",\"env\":\"demo\"}}"
```

Query by API:

```powershell
curl.exe "http://localhost:8080/api/search?q=payment&label=service=billing&limit=10"
```

Metrics:

```powershell
curl.exe http://localhost:8080/api/metrics
```
