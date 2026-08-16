# Engineering standards

## Money and spend control

- **Evaluate approval authority against the invoice total, never against individual
  lines.** A limit check that passes line-by-line lets a submitter split one large
  invoice into small lines and clear it under their own delegated authority. The
  total is the only amount that represents committed spend.

- **Payment idempotency keys must be deterministic**, derived only from stable
  identity — the invoice ID and attempt number. Anything time- or random-based
  produces a fresh key on retry, and the gateway charges twice.

- **Represent monetary amounts as `int64` whole cents.** Never `float64`. Name the
  field `...Cents` so the unit is unambiguous at every call site.
