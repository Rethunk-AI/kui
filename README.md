# KUI

Web-based KVM User Interface for users who prefer a UI over CLI. Connects to libvirt hosts for VM lifecycle, provisioning, and monitoring.

## Docs

- [PRD](docs/prd.md)
- [Architecture](docs/prd/architecture.md)
- [Stack](docs/prd/stack.md)

## Build

```bash
go build -o bin/ ./cmd/...
go test ./...
go vet ./...
```

## Contributing

Open an issue or merge request. See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

[MIT](LICENSE)
