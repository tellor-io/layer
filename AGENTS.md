# Agent Guide — Tellor Layer

Cosmos SDK chain (Go 1.24). Deterministic facts only; session state lives in
`HANDOFF.md` and `docs/sessions/`.

## Commands

- Build: `make build`; install binary: `make install`.
- Fast tests: `make test` (`go test ./... -short`).
- Lint: `make lint` (golangci-lint; 15m budget). Fix mode: `make lint-fix`; single module: `make lint-folder-fix FOLDER="x/mint"`.
- Canonical pre-done check: `make lint && make test`.
- E2E: `make e2e` (docker-based live chain, ~30 min). Do not run by default; name it as skipped coverage.
- Proto changes: `make proto-gen`; mocks: `make mock-gen`.
- Local devnet: `make get-heighliner && make local-image && make get-localic && make local-devnet` (RPC :26657, LCD :1317, gRPC :9090).
- Never push. Never commit unless explicitly asked.

## Consensus-Critical Areas

Diffs touching these modules require `/skill:consensus-safety-review` before
they are considered done: `x/reporter` (validator power, stake-to-power,
jailing, rewards), `x/bridge` (attestation slashing, valset signatures),
`x/oracle` (aggregation, liveness), `x/dispute` (vote tally, verdicts,
slashing). `x/mint` and `x/registry` are lower risk but still state-machine
code — evidence over assertion for any behavior claim.

## Session Workflow

- On session start: read `HANDOFF.md` if present (`/pickup` in Pi); run
  `/skill:branch-triage` when returning after a gap — this repo carries many
  local branches.
- Durable logs in `docs/sessions/`; plans in `docs/plans/`; review findings
  in `docs/reviews/` (create on first use). End meaningful sessions with
  `/handoff`.
