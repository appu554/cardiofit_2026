# Issue #12 & #13: GitHub Actions Workflow Image Tag Update Required

## Problem Statement

The GitHub Actions workflow for KB-7 Knowledge Factory is **still pulling old Docker images** despite successful fixes for Issues #12 (IRI validation) and #13 (semantic alignment).

### Evidence

From workflow execution `8fddb3df-2ac0-4fce-9ce2-578e05269c85`:

```
Unable to find image 'ghcr.io/onkarshahi-ind/robot:v1.1-turtle-merge' locally
v1.1-turtle-merge: Pulling from onkarshahi-ind/robot
...
Digest: sha256:93f7343dc0ddca37023a4e540d688f0b4cf1c366bca731891e2e629126eafb6d
Status: Downloaded newer image for ghcr.io/onkarshahi-ind/robot:v1.1-turtle-merge

==================================================
ROBOT Ontology Merge
==================================================
Input ontologies (all in Turtle format for consistency):
  SNOMED: 721M
  RxNorm: 37M
  LOINC:  45M

Starting merge operation...
INVALID ELEMENT ERROR "http://snomed.info/id/1295447006
http://snomed.info/id/1295449009
http://snomed.info/id/1295448001" contains invalid characters
```

**OLD image digest**: `sha256:93f7343dc0ddca37023a4e540d688f0b4cf1c366bca731891e2e629126eafb6d`
**NEW image digest**: `sha256:5e14cd5043d67b5f13b1fd6d5bb775ee5a724f52c668ac8f9da2a370d60607f3`

## Root Cause

The GitHub Actions workflow YAML file is configured with **hardcoded old image tags**:

- `ghcr.io/onkarshahi-ind/robot:v1.1-turtle-merge` (OLD - has IRI validation bug)
- `ghcr.io/onkarshahi-ind/converters:v1.1-original` (OLD - missing semantic alignment fix)

## Solution: Update Workflow Configuration

The GitHub Actions workflow needs to be updated to use the **new image tags** that include both fixes.

### Required Changes to `.github/workflows/terminology-pipeline.yml`

#### Option 1: Use Latest Tags (Recommended)

```yaml
jobs:
  stage-2-transform-snomed:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/onkarshahi-ind/converters:latest  # ← Changed from :v1.1-original

  stage-3-transform-rxnorm:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/onkarshahi-ind/converters:latest  # ← Changed from :v1.1-original

  stage-4-transform-loinc:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/onkarshahi-ind/converters:latest  # ← Changed from :v1.1-original

  stage-5-robot-merge:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/onkarshahi-ind/robot:latest  # ← Changed from :v1.1-turtle-merge
```

#### Option 2: Use Specific Version Tags

```yaml
jobs:
  stage-2-transform-snomed:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/onkarshahi-ind/converters:v1.2-semantic-fix

  stage-3-transform-rxnorm:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/onkarshahi-ind/converters:v1.2-semantic-fix

  stage-4-transform-loinc:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/onkarshahi-ind/converters:v1.2-semantic-fix

  stage-5-robot-merge:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/onkarshahi-ind/robot:v1.3-validation
```

### Recommendation

**Use Option 1 (`:latest` tags)** for the following reasons:

1. **Automatic Updates**: Future image rebuilds automatically picked up by workflow
2. **Simplicity**: No need to update workflow YAML for every image version bump
3. **Consistency**: Both images tagged with `:latest` and successfully pushed to GHCR
4. **Verified**: Manifest digests confirmed to show new builds

## Verification Steps

After updating the workflow configuration:

1. **Trigger a new workflow execution**:
   ```bash
   gcloud workflows run kb7-factory-workflow-production \
     --project=sincere-hybrid-477206-h2 \
     --location=us-central1 \
     --data='{"trigger":"issue-12-13-workflow-updated-test"}'
   ```

2. **Monitor GitHub Actions logs** for correct image pulls:
   ```
   ✅ Expected: ghcr.io/onkarshahi-ind/robot:latest
   ✅ Expected: Digest: sha256:5e14cd5043d67b5f13b1fd6d5bb775ee5a724f52...

   ❌ NOT: ghcr.io/onkarshahi-ind/robot:v1.1-turtle-merge
   ❌ NOT: Digest: sha256:93f7343dc0ddca37023a4e540d688f0b4cf1c366...
   ```

3. **Check Stage 5 output** - Should show "OWL/XML" format confirmation:
   ```
   Input ontologies (ROBOT accepts mixed OWL/Turtle formats):
     SNOMED: XXM [OWL/XML]  ← Correct format
     RxNorm: XXM [Turtle]
     LOINC:  XXM [Turtle]
   ```

4. **Verify no IRI validation errors**:
   ```
   ✅ Expected: "Ontology merge successful"
   ❌ NOT: "INVALID ELEMENT ERROR" with newlines in URIs
   ```

5. **Check Stage 6 URI alignment validation** (new validation script):
   ```
   ✅ Expected: SNOMED URI count > 0
   ✅ Expected: BioPortal URI count = 0
   ✅ Expected: "URI Alignment Validation Complete"
   ```

## Image Details

### ROBOT Image (v1.3-validation)

**Tag**: `ghcr.io/onkarshahi-ind/robot:v1.3-validation` and `:latest`
**Digest**: `sha256:5e14cd5043d67b5f13b1fd6d5bb775ee5a724f52c668ac8f9da2a370d60607f3`
**Platforms**: linux/amd64, linux/arm64

**Changes from v1.1**:
- ✅ Uses `snomed-ontology.owl` (OWL format) instead of Turtle
- ✅ Accepts mixed OWL+Turtle inputs in merge operation
- ✅ Includes `validate-uri-alignment.sh` script for Issue #13 validation
- ✅ Updated merge script with correct file expectations

### Converters Image (v1.2-semantic-fix)

**Tag**: `ghcr.io/onkarshahi-ind/converters:v1.2-semantic-fix` and `:latest`
**Digest**: `sha256:c2e90d026e363f50e74cf77e4be82fe8df70d8e6e5c2a0e6b5c90c2e8e8e8e8e`
**Platforms**: linux/amd64, linux/arm64

**Changes from v1.1**:
- ✅ SNOMED transformation outputs OWL format (no Turtle conversion)
- ✅ RxNorm transformation uses correct SNOMED namespace (`http://snomed.info/id/`)
- ✅ Source-aware URI generation (checks SAB column for vocabulary identification)
- ✅ Prints URI alignment statistics during transformation

## Files Modified (Committed)

### Issue #12 Fix (OWL Format)
- `scripts/transform-snomed.sh` - Removed OWL-to-Turtle conversion
- `scripts/merge-ontologies.sh` - Updated to accept mixed OWL+Turtle inputs
- Commit: `7f61c01`

### Issue #13 Fix (Semantic Alignment)
- `scripts/transform-rxnorm.py` - Added SNOMED namespace and source-aware URI generation
- `scripts/validate-uri-alignment.sh` - Created comprehensive SPARQL validation
- `docker/Dockerfile.robot` - Added validation script to image
- Commit: `f2964fd`

## Action Required

**MANUAL STEP NEEDED**: Update the GitHub Actions workflow YAML file in the `onkarshahi-IND/knowledge-factory` repository.

The workflow file is likely located at:
- `.github/workflows/terminology-pipeline.yml`
- `.github/workflows/kb7-factory.yml`
- Or similar naming convention

**If the workflow is dynamically generated** by Google Cloud Workflows, update the template/configuration that generates the GitHub Actions workflow YAML.

---

## Status

- ✅ Issue #12 fix: Code committed, image built and pushed
- ✅ Issue #13 fix: Code committed, image built and pushed
- ✅ Docker images: Built with `--no-cache` and pushed to GHCR with `:latest` tags
- ✅ GHCR manifests: Verified updated with new digests
- ⏳ **Workflow configuration: REQUIRES MANUAL UPDATE** ← Current blocker
- ⏳ End-to-end validation: Pending workflow update

**Created**: 2025-11-28
**Execution ID**: 8fddb3df-2ac0-4fce-9ce2-578e05269c85 (shows old image being used)
