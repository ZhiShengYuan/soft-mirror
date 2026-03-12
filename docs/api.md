# API Reference

Base URL: `http://localhost:8080`

All API endpoints are under the `/api/v1` prefix. JSON responses use `application/json` content type. Error responses follow a consistent format (see [Error responses](#error-responses)).

---

## Endpoints

### Health check

```
GET /healthz
```

Returns server health status. No authentication required.

**Response** `200 OK`

```json
{
  "status": "ok"
}
```

**Example**

```bash
curl http://localhost:8080/healthz
```

---

### List versions

```
GET /api/v1/programs/:name/versions
```

Returns all versions of a program with their available platforms.

| Parameter | In | Required | Description |
|---|---|---|---|
| `name` | path | yes | Program name (`^[a-zA-Z0-9_-]+$`, max 128 chars) |

**Response** `200 OK`

```json
{
  "program": "myapp",
  "versions": [
    {
      "version": "1.2.3",
      "platforms": [
        { "OS": "linux", "Arch": "amd64" },
        { "OS": "darwin", "Arch": "arm64" }
      ]
    }
  ]
}
```

Returns an empty `versions` array if the program has no versions.

**Example**

```bash
curl http://localhost:8080/api/v1/programs/myapp/versions
```

---

### Auto-download

```
GET /api/v1/programs/:name/download
```

Detects the client's OS and architecture from the `User-Agent` header, resolves the requested version, and issues a `302` redirect to the direct download URL.

| Parameter | In | Required | Default | Description |
|---|---|---|---|---|
| `name` | path | yes | | Program name |
| `version` | query | no | `latest` | Version query: `latest`, exact version (`1.2.3`, `v1.2.3`), or semver constraint (`^1.0`, `~1.2`, `>=1.0 <2.0`) |
| `os` | query | no | _(auto-detected)_ | Override OS: `linux`, `darwin`, `windows` |
| `arch` | query | no | _(auto-detected)_ | Override architecture: `amd64`, `arm64` |

**OS auto-detection** from User-Agent:
- `darwin`, `macos`, `macintosh` -> `darwin`
- `windows`, `win` -> `windows`
- `linux` or anything else -> `linux`

**Arch auto-detection** from User-Agent:
- `aarch64`, `arm64` -> `arm64`
- `x86_64`, `amd64`, `x64` or anything else -> `amd64`

**Response** `302 Found` -- redirects to `/api/v1/programs/:name/:version/:os/:arch` with a `Cache-Control: no-cache` header.

**Examples**

```bash
# Auto-detect everything, get latest
curl -LO http://localhost:8080/api/v1/programs/myapp/download

# Specific version and platform
curl -LO "http://localhost:8080/api/v1/programs/myapp/download?version=1.2.3&os=linux&arch=amd64"

# Semver constraint
curl -LO "http://localhost:8080/api/v1/programs/myapp/download?version=^1.0"
```

---

### Direct download

```
GET /api/v1/programs/:name/:version/:os/:arch
```

Downloads a specific binary. Supports `ETag` conditional requests and HTTP range requests.

| Parameter | In | Required | Description |
|---|---|---|---|
| `name` | path | yes | Program name |
| `version` | path | yes | Exact semver version (e.g. `1.2.3` or `v1.2.3`) |
| `os` | path | yes | `linux`, `darwin`, or `windows` |
| `arch` | path | yes | `amd64` or `arm64` |

**Response headers**
- `ETag` -- `"{modtime_unix}-{size}"` format
- `Cache-Control: public, max-age=86400`
- `Content-Disposition: attachment; filename="{name}"` (or `{name}.exe` for windows)

**Response** `200 OK` -- binary file stream (supports `If-None-Match`, `If-Modified-Since`, and `Range` request headers via `http.ServeContent`).

**Example**

```bash
curl -O http://localhost:8080/api/v1/programs/myapp/1.2.3/linux/amd64
```

---

### Upload binary

```
PUT /api/v1/programs/:name/:version/:os/:arch
```

Uploads a binary for the given program, version, OS, and architecture. **Requires HMAC authentication.**

The binary content is sent as the raw request body (not multipart form).

| Parameter | In | Required | Description |
|---|---|---|---|
| `name` | path | yes | Program name |
| `version` | path | yes | Semver version |
| `os` | path | yes | `linux`, `darwin`, or `windows` |
| `arch` | path | yes | `amd64` or `arm64` |

**Request body:** Raw binary content. Maximum size is configured by `max_upload_size` (default 512 MiB).

**Response** `201 Created`

```json
{
  "message": "uploaded successfully",
  "program": "myapp",
  "version": "1.2.3",
  "os": "linux",
  "arch": "amd64"
}
```

**Example**

```bash
# See "HMAC signing" section below for how to compute the signature
curl -X PUT \
  -H "Authorization: HMAC ${SIGNATURE}" \
  -H "X-Timestamp: ${TIMESTAMP}" \
  -H "X-Nonce: ${NONCE}" \
  --data-binary @myapp \
  http://localhost:8080/api/v1/programs/myapp/1.2.3/linux/amd64
```

---

### Delete binary

```
DELETE /api/v1/programs/:name/:version/:os/:arch
```

Deletes a single binary for a specific platform. Empty parent directories are cleaned up automatically. **Requires HMAC authentication.**

| Parameter | In | Required | Description |
|---|---|---|---|
| `name` | path | yes | Program name |
| `version` | path | yes | Semver version |
| `os` | path | yes | `linux`, `darwin`, or `windows` |
| `arch` | path | yes | `amd64` or `arm64` |

**Response** `200 OK`

```json
{
  "message": "deleted"
}
```

**Example**

```bash
curl -X DELETE \
  -H "Authorization: HMAC ${SIGNATURE}" \
  -H "X-Timestamp: ${TIMESTAMP}" \
  -H "X-Nonce: ${NONCE}" \
  http://localhost:8080/api/v1/programs/myapp/1.2.3/linux/amd64
```

---

### Delete version

```
DELETE /api/v1/programs/:name/:version
```

Deletes an entire version directory and all platform binaries within it. **Requires HMAC authentication.**

| Parameter | In | Required | Description |
|---|---|---|---|
| `name` | path | yes | Program name |
| `version` | path | yes | Semver version |

**Response** `200 OK`

```json
{
  "message": "version deleted"
}
```

**Example**

```bash
curl -X DELETE \
  -H "Authorization: HMAC ${SIGNATURE}" \
  -H "X-Timestamp: ${TIMESTAMP}" \
  -H "X-Nonce: ${NONCE}" \
  http://localhost:8080/api/v1/programs/myapp/1.2.3
```

---

### Web pages

These serve HTML pages for browsing in a browser. They are not part of the JSON API.

| Route | Description |
|---|---|
| `GET /` | Index page listing all programs with latest version and platform info |
| `GET /programs/:name` | Program detail page showing all versions and download links |

---

## HMAC signing protocol

Upload and delete endpoints require HMAC-SHA256 authentication. Every authenticated request must include three headers:

| Header | Description |
|---|---|
| `Authorization` | `HMAC <hex-encoded-signature>` |
| `X-Timestamp` | Current Unix timestamp (seconds) |
| `X-Nonce` | Unique random string per request |

### Signing algorithm

1. Compute the SHA-256 hash of the request body (empty string for DELETE), hex-encoded.
2. Build the string to sign by joining these five fields with newlines (`\n`):
   ```
   METHOD\nPATH\nTIMESTAMP\nNONCE\nBODY_HASH
   ```
   - `METHOD` -- HTTP method (e.g. `PUT`, `DELETE`)
   - `PATH` -- request path (e.g. `/api/v1/programs/myapp/1.2.3/linux/amd64`)
   - `TIMESTAMP` -- same value as the `X-Timestamp` header
   - `NONCE` -- same value as the `X-Nonce` header
   - `BODY_HASH` -- hex-encoded SHA-256 of the request body
3. Compute HMAC-SHA256 of the string to sign using the shared secret.
4. Hex-encode the result and send it in the `Authorization` header as `HMAC <hex>`.

### Validation rules

- **Clock drift** -- the timestamp must be within `hmac_max_drift` of the server's clock (default 5 minutes).
- **Replay protection** -- each nonce can only be used once within the drift window. Reusing a nonce returns a 401 error.
- **Constant-time comparison** -- signature verification uses `crypto/hmac.Equal` to prevent timing attacks.

### Example: signing a PUT upload in bash

```bash
SECRET="your-secret"
METHOD="PUT"
PATH="/api/v1/programs/myapp/1.2.3/linux/amd64"
TIMESTAMP=$(date +%s)
NONCE=$(openssl rand -hex 16)
BODY_HASH=$(sha256sum myapp | awk '{print $1}')

STRING_TO_SIGN="${METHOD}\n${PATH}\n${TIMESTAMP}\n${NONCE}\n${BODY_HASH}"
SIGNATURE=$(printf '%b' "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $NF}')

curl -X PUT \
  -H "Authorization: HMAC ${SIGNATURE}" \
  -H "X-Timestamp: ${TIMESTAMP}" \
  -H "X-Nonce: ${NONCE}" \
  --data-binary @myapp \
  "http://localhost:8080${PATH}"
```

### Example: signing a DELETE in bash

```bash
SECRET="your-secret"
METHOD="DELETE"
PATH="/api/v1/programs/myapp/1.2.3/linux/amd64"
TIMESTAMP=$(date +%s)
NONCE=$(openssl rand -hex 16)
BODY_HASH=$(printf '' | sha256sum | awk '{print $1}')

STRING_TO_SIGN="${METHOD}\n${PATH}\n${TIMESTAMP}\n${NONCE}\n${BODY_HASH}"
SIGNATURE=$(printf '%b' "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $NF}')

curl -X DELETE \
  -H "Authorization: HMAC ${SIGNATURE}" \
  -H "X-Timestamp: ${TIMESTAMP}" \
  -H "X-Nonce: ${NONCE}" \
  "http://localhost:8080${PATH}"
```

---

## Error responses

All errors return a JSON object with a single `error` field:

```json
{
  "error": "description of what went wrong"
}
```

### Common HTTP status codes

| Status | Meaning |
|---|---|
| `400 Bad Request` | Invalid program name, version, OS, arch, or body exceeds size limit |
| `401 Unauthorized` | HMAC authentication failed (missing headers, invalid signature, expired timestamp, replayed nonce) |
| `404 Not Found` | Program, version, or binary does not exist |
| `500 Internal Server Error` | Unexpected server-side failure |

### Validation constraints

| Field | Rules |
|---|---|
| Program name | `^[a-zA-Z0-9_-]+$`, 1--128 characters |
| Version | Valid semver, with or without `v` prefix |
| OS | `linux`, `darwin`, or `windows` |
| Architecture | `amd64` or `arm64` |
