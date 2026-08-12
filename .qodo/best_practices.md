# Engineering standards

Rules reviewers are expected to enforce on every PR. Each one exists because
breaking it has cost us money or an audit finding.

## Money

- **Represent all monetary amounts as `int64` whole cents** (Go) or `number` of
  whole cents (TypeScript). Never `float64`, `float32`, or a decimal string
  parsed into a float. Name the field `...Cents` so the unit is unambiguous at
  every call site.
- **Never round in a loop.** Splitting an amount across GL codes, tax
  components, or entities must use a largest-remainder allocation whose output
  sums to exactly the input. A per-item `round()` drifts and the allocation
  stops reconciling against the invoice.
- **Currency is part of the amount.** Never compare, add, or total amounts
  without confirming they share a `currencyCode`.

## Approval and spend control

- **Evaluate approval authority against the invoice total, never against
  individual lines.** A limit check that passes line-by-line lets a submitter
  split one large invoice into small lines and clear it under their own
  delegated authority. The total is the only amount that represents committed
  spend.
- **Auto-approval must record an approval step** on the chain, with
  `auto: true` and the deciding identity. A state change with no audit row is
  an audit finding, and the auto path is exactly the one with no human to ask.
- **Approval state transitions are read-modify-write and must run in a
  transaction.** Two approvers acting simultaneously must not both claim the
  same pending step.

## Payments

- **Idempotency keys must be deterministic, derived only from stable identity**
  — invoice ID, entity ID, payment run ID. Never from `time.Now()`, a random
  value, or a retry counter. A key that changes between attempts turns every
  retry of an ambiguous timeout into a second real payment.
- **Book to the ERP before releasing funds.** The ERP is the system of record;
  we never report a payment we could not book against it.
- **Never swallow an integration error.** No `_ = err` on an ERP, gateway, or
  batch-commit call. Partial success reported as success is worse than a
  failure.

## Multi-entity isolation

- **Every query touching entity-scoped data filters on both `userId` and
  `entityId`.** Applying a status or date filter must never replace the
  isolation predicates — build queries by chaining `Where` onto a base query
  that already carries them, never by rebuilding the query from scratch inside
  a conditional.

## Cross-stack contracts

- **A Go struct's `json` tags and its TypeScript mirror must match field for
  field.** When a Go type in `api/services/` changes shape, the mirroring type
  in `frontend/src/types/` changes in the same PR. `handleResponse<T>()` casts
  without validating, so a mismatch is silent at compile time and surfaces as
  `undefined`/`NaN` in the UI.
- Keep the mirroring file named after its Go source and note the source path
  in a header comment, so a reviewer can find the other half.

## Tests

- Cover the **boundary** of any threshold, not just a value on each side of it:
  at the limit, one cent under, one cent over.
- Cover the **multi-item** shape of anything that aggregates. A rule verified
  only against single-line inputs is not verified.
- Retry paths need a test where the **first attempt fails**. A happy-path test
  never exercises the retry.
