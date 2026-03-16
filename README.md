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

Libvirt bindings require CGO and `libvirt-dev` headers. If those headers are unavailable, build and test the project without libvirt-enabled files using:

```bash
go build ./...
go test ./...
go vet ./...
```

Run the libvirt connector integration test against `test:///default` with:

```bash
go test -tags libvirt ./internal/libvirtconn/...
```

## Contributing

Open an issue or merge request. See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

[MIT](LICENSE)
