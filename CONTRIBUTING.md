# Contributing Guide

Thank you for your interest in contributing to eBPF Firewall! This guide will help you get started with the contribution process.

## Table of Contents

- [Getting Started](#getting-started)
  - [Fork and Clone](#fork-and-clone)
  - [Development Setup](#development-setup)
  - [Create a Branch](#create-a-branch)
- [Development Workflow](#development-workflow)
  - [Code Style](#code-style)
  - [Commit Convention](#commit-convention)
  - [Sync with Upstream](#sync-with-upstream)
  - [Submit Pull Request](#submit-pull-request)

## Getting Started

### Fork and Clone

1. Fork the [eBPF Firewall](https://github.com/danger-dream/ebpf-firewall) repository
2. Clone your fork:

```bash
git clone https://github.com/YOUR_USERNAME/ebpf-firewall.git
cd ebpf-firewall
```

### Development Setup

1. Install dependencies:

```bash
apt update && apt upgrade
apt install llvm clang libbpf-dev build-essential linux-headers-$(uname -r)

# If asm/types.h reference error occurs:
ln -s /usr/include/x86_64-linux-gnu/asm /usr/include/asm
```

2. Build frontend:

```bash
cd web
npm install
npm run build
```

3. Generate eBPF object files:

```bash
cd ../internal/ebpf
go generate
```

### Create a Branch

Create a new branch for your contribution:

```bash
git checkout -b your-branch-name
```

## Development Workflow

### Code Style

- Go code should follow the standard Go style guidelines
- TypeScript code should follow the project's ESLint configuration
- C code (eBPF) should follow the Linux kernel coding style

### Commit Convention

```
type(scope): body
```

#### Type

Must be one of the following:

- `feat`: A new feature (minor version bump)
- `fix`: A bug fix (patch version bump)
- `perf`: A code change that improves performance
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `docs`: Documentation only changes
- `chore`: Changes to the build process or auxiliary tools
- `build`: Changes that affect the build system or external dependencies
- `ci`: Changes to CI configuration files and scripts
- `revert`: Reverts a previous commit
- `improvement`: Improvements to existing features

Note: Adding `BREAKING CHANGE:` in commit body will trigger a major version bump.

#### Scope

Must be one of the following:

- `ebpf`: Changes to eBPF programs
- `usercomm`: Changes to user space communication
- `dataproc`: Changes to data processing logic
- `frontend`: Changes to frontend components
- `ui`: Changes to UI/UX
- `docs`: Changes to documentation
- `build`: Changes to build configurations
- `deploy`: Changes to deployment process
- `other`: Other changes

### Sync with Upstream

Keep your fork up to date:

```bash
git remote add upstream https://github.com/danger-dream/ebpf-firewall.git
git fetch upstream
git merge upstream/main
```

### Submit Pull Request

1. Push your changes to your fork
2. Go to the original repository and create a Pull Request
3. Follow the Pull Request template
4. Wait for review and address any feedback

## Additional Notes

- All PRs should target the `dev` branch
- Include tests for new features
- Update documentation when necessary
- Ensure all tests pass before submitting
