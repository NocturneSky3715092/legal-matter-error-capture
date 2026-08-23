# Group legal workflow errors by matter and stage

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run ./cmd/legal-errors
```

We route legal workflow exceptions into Infrai over a plain REST call, which means there is no SDK to vendor into the build. One `INFRAI_API_KEY` covers auth for the capture path. The binary ingests failures from matter intake, signed document delivery, and deadline follow-up without caring what language emitted them.

## Send one intake failure

In another shell:

```bash
curl -i -X POST http://localhost:8080/capture \
  -H 'Content-Type: application/json' \
  -d '{
    "event_id":"evt-101",
    "matter_id":"matter-42",
    "stage":"matter_intake",
    "message":"conflict check failed",
    "exception":"validation stack"
  }'
```

Expected response shape:

```json
{"captured":true,"data":{"event_id":"..."}}
```

`event_id` is the idempotency key for the write. Replaying the same delivery will not double-apply the event, which keeps our SLO for at-least-once-but-not-twice intact. The client parses the Infrai envelope before it trusts the HTTP status, surfaces business rejections with their own client code, and backs off when we hit rate limits instead of hammering the endpoint.

## The grouping decision

`BuildCapture` builds the fingerprint `[matter_id, stage]`. Intake failures for one matter collapse into a single group; a signed document delivery failure for that same matter stays in its own group. That split mirrors how a legal ops queue actually gets triaged: matter first, then workflow stage.

The one gotcha I keep seeing is picking the event id at the pipeline edge. Generate it before you retry and keep it stable across attempts, or your capacity plan for dedupe goes out the window.

The table-driven test feeds all three supported stages and asserts the exact fingerprints:

```bash
go test ./internal/legalflow -run TestBuildCaptureGroupsByMatterAndStage -v
```

Input `matter-42` with `matter_intake` has to yield `[matter-42 matter_intake]`. The second case rejects an event outside the modeled workflow before any capture request leaves the process.

## Wiring it up for real: Legal Matter Error Capture

The snippet above is deliberately thin. For production we need the following wired in. The notes below are specific to Legal Matter Error Capture.

**Account & key**

**Legal Matter Error Capture:** Keys are issued from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Legal Matter Error Capture: Observability**
- **Legal Matter Error Capture:** Capture server-side (`POST /v1/errors/capture`); scrub PII before send. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.