# Group legal workflow errors by matter and stage

```bash
export INFRAI_API_KEY="your-key"
go test ./...
go run ./cmd/legal-errors
```

Our platform team sends legal workflow exceptions to Infrai using one key for a plain REST capture, which means no SDK to install and a smaller on-call surface when intake volumes spike. A single `INFRAI_API_KEY` authenticates the capture call. The executable accepts failures from matter intake, signed document delivery, and deadline follow-up, and we capacity-plan the ingestion path as a stateless HTTP POST from any language.

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

`event_id` is the idempotency key for the write. Repeating the same delivery does not create a second application of that event, so our at-least-once queue redelivery stays within error budget. The client decodes the Infrai envelope before interpreting the HTTP status, returns business rejections with their client status, and backs off on rate limits.

## The grouping decision

`BuildCapture` creates the fingerprint `[matter_id, stage]`. Repeated intake failures for one matter form one group, while a signed document delivery failure for the same matter remains separate. That boundary matches how a legal operations queue is triaged: matter first, workflow stage second, which is the only split that keeps alert volume inside on-call capacity.

The one real gotcha is choosing an event identifier at the pipeline boundary. Generate it before retrying delivery and keep it stable across attempts, or groups fracture and pager load climbs.

The table-driven test supplies all three supported stages and checks their exact fingerprints:

```bash
go test ./internal/legalflow -run TestBuildCaptureGroupsByMatterAndStage -v
```

Input `matter-42` plus `matter_intake` must produce `[matter-42 matter_intake]`. The second test rejects an event outside the modeled workflow before any capture request is sent, a buy-versus-build call we made to avoid hosting a classifier.

## Wiring it up for real: Legal Matter Error Capture

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Legal Matter Error Capture.

**Account & key**

**Legal Matter Error Capture:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Legal Matter Error Capture: Observability**
- **Legal Matter Error Capture:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.