# Klar Configuration Formats

The config formats that are currently implemented are:

- [Glas Manifests (`glas.pack`)](#glas-manifest)
- [Glas Lockfiles (`glas.lock`)](#glas-lockfile)
- [Klar Build Config (`klar.build`)](#klar-build-config)
- [Klar Formatter Config (`klar.fmt`)](#klar-formatter-config)

Expect more formats to be added in the future (Klar CLI config `klar.conf` and Klar lint configs `klar.lint`).

## Glas Manifest (`glas.pack`)

- **Definitions:** [glaspack/glaspack.go](./glaspack/glaspack.go)
- **Language:** Klon

## Glas Lockfile (`glas.lock`)

- **Definitions:** [glaslock/lockfile.go](./glaslock/lockfile.go)
- **Language:** Custom line-based format
- **Parser:** [glaslock/parse.go](./glaslock/parse.go)

## Klar Build Config (`klar.build`)

- **Definitions:** [klarbuild/config.go](./klarbuild/config.go)
- **Language:** Klon

## Klar Formatter Config (`klar.fmt`)
