# Production CI/CD Best Practices: Docker Image Tagging

**Date**: 2025-11-28
**Context**: Issues #12 & #13 Resolution - Moving to Production-Ready CI/CD

---

## Summary

This document captures critical lessons learned about production CI/CD practices, specifically around Docker image tagging strategies and their impact on deployment reliability, audit trails, and rollback capabilities.

---

## The Problem with `:latest`

### ❌ **What We Did Wrong**

Our initial workflow used `:latest` tags for Docker images:

```yaml
docker run --rm \
  -v $(pwd)/output:/output \
  ghcr.io/onkarshahi-ind/snomed-toolkit:latest  # ❌ BAD IN PRODUCTION
```

### ⚠️ **Why This is Dangerous**

| Issue | Impact | Example |
|-------|--------|---------|
| **No Reproducibility** | Can't tell which version is deployed | "It worked yesterday!" |
| **Cache Confusion** | Docker may not pull updates | Local works, prod fails |
| **Rollback Impossible** | Can't revert to specific version | "Which image was working?" |
| **No Audit Trail** | Can't trace issues to commits | "When did this break?" |
| **Non-Deterministic** | Same tag, different content over time | Deployment roulette |

### 🎯 **Real-World Failure Scenario**

```
Developer: "Deploy snomed-toolkit:latest"
   ↓
Pipeline: Uses cached :latest (2 weeks old)
   ↓
Production: Runs old code with known bug
   ↓
Incident: "But it works on my machine!"
```

---

## ✅ The Production-Ready Solution

### **Immutable Tags with Git SHA**

```yaml
# ✅ CORRECT: Immutable, traceable, rollback-able
docker run --rm \
  -v $(pwd)/output:/output \
  ghcr.io/onkarshahi-ind/snomed-toolkit:365a286  # Git SHA
```

### **Why This Works**

| Benefit | Description | Example |
|---------|-------------|---------|
| **Immutable** | Tag never changes content | `365a286` always = commit 365a286 |
| **Traceable** | Direct link to source code | `git show 365a286` |
| **Rollback-able** | Instant revert to any version | Deploy `233fa36` (previous SHA) |
| **Audit Trail** | Full deployment history | "Issue started with `f2964fd`" |
| **Deterministic** | Same tag = same code always | No surprises |

---

## 📋 Implementation Guide

### 1. Build Script with Git SHA Tagging

Created: [`build-and-tag-images.sh`](./build-and-tag-images.sh)

```bash
#!/bin/bash
GIT_SHA=$(git rev-parse --short HEAD)

# Tag with SHA (immutable)
docker build \
  --tag ghcr.io/owner/snomed-toolkit:$GIT_SHA \
  --tag ghcr.io/owner/snomed-toolkit:main \
  .
```

**Usage**:
```bash
./build-and-tag-images.sh --push
# Builds and tags: snomed-toolkit:c0a8b18
```

### 2. Workflow Configuration (Required Changes)

#### **A. Enable Full Git History**

```yaml
- name: Checkout Code
  uses: actions/checkout@v4
  with:
    fetch-depth: 0  # ✅ Full history, not shallow clone
```

**Why**: Shallow clones (`fetch-depth: 1`) don't have commit history for SHA tagging.

#### **B. Build and Tag with SHA**

```yaml
- name: Build Docker Images
  run: |
    GIT_SHA=$(git rev-parse --short HEAD)

    docker build \
      --file docker/Dockerfile.snomed-toolkit \
      --tag ghcr.io/${{ github.repository_owner }}/snomed-toolkit:$GIT_SHA \
      --push \
      .
```

#### **C. Use SHA Tags in Pipeline**

```yaml
- name: Transform SNOMED
  run: |
    GIT_SHA=$(git rev-parse --short HEAD)

    docker run --rm \
      -v $(pwd)/output:/output \
      ghcr.io/${{ github.repository_owner }}/snomed-toolkit:$GIT_SHA
```

---

## 🔍 Shallow Clone Issues

### **The Problem**

GitHub Actions defaults to shallow clones (`fetch-depth: 1`) for speed:

```yaml
- uses: actions/checkout@v4  # Only grabs latest commit!
```

### **Symptoms**

```bash
$ git rev-list --count HEAD
1  # Only 1 commit visible!

$ git log --oneline
c0a8b18 Latest commit  # Missing all history
```

### **Impact**

- **Can't calculate git describe**: Needs tags in history
- **Can't detect changed files**: `git diff` across commits fails
- **Can't reference SHAs**: Previous commits not available

### **Fix**

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0  # ✅ Get full history
```

```bash
$ git rev-list --count HEAD
50  # All commits available

$ git log --oneline -5
c0a8b18 Documentation
365a286 IRI sanitization fix
233fa36 Workflow updates
...
```

---

## 🏭 Production Deployment Pattern

### **GitOps Flow (ArgoCD/Flux)**

```
1. Developer commits code
   └─> Git SHA: abc123d

2. CI builds and tags image
   └─> ghcr.io/owner/app:abc123d

3. CI updates manifest
   └─> k8s/deployment.yaml: image: app:abc123d

4. ArgoCD detects manifest change
   └─> Deploys app:abc123d to cluster

5. Deployment complete
   └─> Audit: "Who deployed abc123d? When? Why?"
```

### **Rollback Scenario**

```
Incident detected at 14:30
   └─> Check git: abc123d deployed at 14:15
   └─> Check logs: Error started at 14:16
   └─> Conclusion: abc123d is bad
   └─> Rollback: Update manifest to xyz789a (previous SHA)
   └─> ArgoCD deploys xyz789a
   └─> Service restored in < 2 minutes
```

---

## 📊 Comparison Matrix

| Aspect | `:latest` | Git SHA |
|--------|-----------|---------|
| **Reproducibility** | ❌ None | ✅ Perfect |
| **Audit Trail** | ❌ No link to code | ✅ Direct git commit |
| **Rollback** | ❌ Impossible | ✅ Instant |
| **Cache Issues** | ❌ Frequent | ✅ Never |
| **Production Ready** | ❌ No | ✅ Yes |
| **Used By** | Dev/Test | Netflix, GitLab, Google |

---

## 🎓 Key Learnings

### 1. **Immutability is Critical**

> **Production Principle**: Once an artifact is created, it never changes.

- Docker image `app:abc123d` contains exact code from commit `abc123d`
- Deploying `abc123d` today = deploying `abc123d` in 6 months
- No "tag reuse" ever

### 2. **Traceability Enables Debugging**

> **When an incident occurs, you need to know EXACTLY what code is running.**

```bash
# With SHA tagging:
$ git show abc123d  # Exact code deployed
$ git log abc123d..HEAD  # What's changed since
$ git diff xyz789a abc123d  # What broke
```

### 3. **Rollbacks Must Be Fast**

> **Every minute of downtime costs money.**

- With SHA tags: Change manifest, redeploy (< 2 min)
- With `:latest`: Find old image... if it still exists (> 30 min)

### 4. **CI/CD Must Have Full History**

> **Shallow clones break git-based workflows.**

- Always use `fetch-depth: 0` in production pipelines
- Enables SHA tagging, change detection, version calculation

---

## 🛠️ Migration Checklist

- [ ] ✅ Update build scripts to tag with git SHA
- [ ] ✅ Add `fetch-depth: 0` to all checkout steps
- [ ] ✅ Replace all `:latest` references with `$GIT_SHA`
- [ ] ✅ Update deployment manifests to track SHA tags
- [ ] ✅ Configure ArgoCD/Flux to watch manifest changes
- [ ] ✅ Document rollback procedures
- [ ] ✅ Train team on new workflow
- [ ] ✅ Remove `:latest` tag automation from registry

---

## 📚 References

### Industry Best Practices
- [Google SRE Book - Release Engineering](https://sre.google/sre-book/release-engineering/)
- [GitLab CI/CD Best Practices](https://docs.gitlab.com/ee/ci/docker/best_practices.html)
- [Kubernetes Image Pull Policy](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy)

### Related Issues
- [Issue #12: IRI Sanitization](./ISSUE_12_IRI_SANITIZATION_COMPLETE.md)
- [Issue #13: Semantic Alignment](./ISSUES_12_13_FINAL_RESOLUTION.md)

---

## 🎯 Action Items

### Immediate (This Week)
1. Deploy SHA-tagged images to staging
2. Validate rollback procedures
3. Update documentation

### Short-term (This Month)
1. Migrate all services to SHA tagging
2. Remove `:latest` from production workflows
3. Implement automated manifest updates

### Long-term (This Quarter)
1. Full GitOps adoption with ArgoCD
2. Automated rollback on error detection
3. Deployment analytics dashboard

---

## ✅ Status

| Component | Status | Notes |
|-----------|--------|-------|
| Build Script | ✅ Complete | `build-and-tag-images.sh` |
| IRI Sanitization | ✅ Complete | Commit `365a286` |
| Semantic Alignment | ✅ Complete | Commit `f2964fd` |
| SHA Tagging Strategy | ✅ Documented | This file |
| Workflow Migration | ⏳ Pending | Next step |
| Production Deployment | ⏳ Pending | After validation |

---

**Contributors**: Claude Code (AI Assistant)
**Reviewed By**: [Pending]
**Last Updated**: 2025-11-28
