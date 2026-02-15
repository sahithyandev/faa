# faa

A CLI tool for managing local development environments.

## Installation

```bash
just build
# Binary will be available at bin/faa
```

Or install directly:

```bash
just install
```

## Usage

```bash
faa <subcommand> [options]
```

### Available Commands

#### `setup`
Setup the development environment.

```bash
faa setup [options]
```

#### `daemon`
Start the daemon process.

```bash
faa daemon [options]
```

#### `run`
Run a project.

```bash
faa run [options]
```

#### `status`
Show status of running projects.

```bash
faa status [options]
```

#### `stop`
Stop a running project.

```bash
faa stop [options]
```

#### `routes`
Display route information.

```bash
faa routes [options]
```

## Development

### Build

```bash
just build
```

### Test

```bash
just test
```

### Clean

```bash
just clean
```