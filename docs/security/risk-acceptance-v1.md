# ClarityIT v1.0 Risk Acceptance — Dev-Only Dependency Vulnerabilities

## Document Status
- **Version**: 1.0.0
- **Date**: 2026-06-14
- **Owner**: Platform Engineering
- **Target Remediation Release**: v1.1.0 (post-launch maintenance)

---

## 1. Summary

`npm audit --audit-level=high` reports 6 high-severity vulnerabilities in the ClarityIT frontend development toolchain. All affected packages are in `devDependencies` — none are present in the production Docker image. Production runtime dependencies are clean (`npm audit --omit=dev --audit-level=high` → 0 vulnerabilities).

This document constitutes a formal risk acceptance per the v1.0 Track 7 closure requirements.

## 2. Affected Packages

All are `devDependencies` in `web/package.json`:

| Package | Version Range | Advisory | Severity |
|---------|-------------|----------|----------|
| `esbuild` | 0.17.0 – 0.28.0 | [GHSA-gv7w-rqvm-qjhr](https://github.com/advisories/GHSA-gv7w-rqvm-qjhr) | High |
| `vite` | 4.2.0-beta.0 – 8.0.3 | Depends on vulnerable `esbuild` | High |
| `@vitejs/plugin-react` | 4.0.0-beta.0 – 5.1.4 | Depends on vulnerable `vite` | High |
| `@vitest/mocker` | ≤4.1.0-beta.6 | Depends on vulnerable `vite` | High |
| `vitest` | 1.0.0-beta.0 – 4.1.0-beta.6 | Depends on vulnerable `@vitest/mocker`, `vite` | High |
| `vite-node` | 1.0.0-beta.0 – 5.3.0 | Depends on vulnerable `vite` | High |

## 3. Advisory Details

### GHSA-gv7w-rqvm-qjhr — esbuild Missing Binary Integrity Verification
- **Attack vector**: `NPM_CONFIG_REGISTRY` environment variable can redirect esbuild's binary download to a malicious server when running under Deno's module system
- **Impact**: Remote code execution during build time
- **Affected runtime**: Deno only (not Node.js)
- **Fixed in**: esbuild ≥ 0.28.1 (requires vite ≥ 8.0)

## 4. Why Not Reachable in Production

### Multi-Stage Docker Build
```dockerfile
# Stage 1: Build (includes devDependencies)
FROM node:22-alpine AS build
RUN npm install         # ← devDependencies present here only
RUN npx vite build      # ← build artifact (static HTML/JS/CSS)

# Stage 2: Serve (NO node, NO npm, NO devDependencies)
FROM nginxinc/nginx-unprivileged:alpine
COPY --from=build /app/dist /usr/share/nginx/html
# ← only static files, no JavaScript runtime
```

### Verification
```bash
# Production audit — must be clean
$ npm audit --omit=dev --audit-level=high
found 0 vulnerabilities

# Confirm no node_modules in production image
$ docker exec clarityit-web ls /usr/share/nginx/html
assets/  index.html  vite.svg

$ docker exec clarityit-web which node || echo "node not found"
node not found

$ docker exec clarityit-web which npm || echo "npm not found"
npm not found
```

The production image contains only:
- `nginx` binary (static file server)
- Pre-compiled static assets (HTML, CSS, JS bundles)
- No Node.js runtime
- No npm
- No devDependencies
- No esbuild, vite, vitest, or any build tooling

## 5. Compensating Controls

| Control | Implementation |
|---------|---------------|
| Network isolation | Web container on private Docker network; only port 3000 exposed |
| Non-root execution | nginxinc/nginx-unprivileged runs as uid=101 |
| Read-only filesystem | `read_only: true` in docker-compose |
| No privilege escalation | `no-new-privileges:true` security option |
| Build environment isolation | Builds run on Proxmox LXC container, not developer machines |
| Dependency pinning | `package-lock.json` ensures reproducible installs |
| Regular monitoring | `make audit` runs on every CI/deploy cycle |
| Production audit gate | `make audit-prod` must pass with 0 vulnerabilities |

## 6. Production Dependency Audit Separation

The Makefile now provides two audit targets:

| Target | Scope | Gate Status |
|--------|-------|-------------|
| `make audit-prod` | Production runtime deps only (`--omit=dev`) | **MUST be clean** |
| `make audit` | Full audit including dev deps | Informational; dev findings documented here |

### Production Dependencies (clean)
```
@base-ui/react, @fontsource-variable/geist, class-variance-authority,
clsx, lucide-react, react, react-dom, react-router-dom, shadcn,
tailwind-merge, tw-animate-css
```

### Dev-Only Dependencies (with findings)
```
@tailwindcss/vite, @testing-library/*, @types/*, @vitejs/plugin-react,
esbuild (transitive), jsdom, msw, tailwindcss, typescript,
vite, vitest, vite-node (transitive)
```

## 7. Remediation Plan

| Action | Target Release | Status |
|--------|---------------|--------|
| Upgrade to `vite@8` (fixes esbuild chain) | v1.1.0 | Planned |
| Upgrade `vitest` to v5+ (compatible with vite@8) | v1.1.0 | Planned |
| Re-run `npm audit --audit-level=high` (expect clean) | v1.1.0 | Pending |
| Remove this risk acceptance document | v1.1.0 | Pending |

## 8. Acceptance

This risk acceptance is valid for v1.0.0 only. It must be remediated in v1.1.0 by upgrading the Vite/Vitest toolchain to versions that include the esbuild binary integrity fix.

- **Accepted by**: Platform Engineering
- **Date**: 2026-06-14
- **Review date**: v1.1.0 release planning
