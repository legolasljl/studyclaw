# Build Notes

## 本地測試

```bash
go test ./lib ./web
cd web/studyclaw && npm run build
```

## 本地編譯

```bash
go build -o studyclaw .
```

## Linux 二進位

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o studyclaw .
```

## Docker 映像

```bash
docker build -t ghcr.io/legolasljl/studyclaw:latest .
```
