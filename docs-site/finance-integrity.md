# Finance integrity

Financial correctness is TALLY’s primary architecture driver.

- Money uses exact decimal representations; binary floating point is not suitable for financial amounts.
- Posted or otherwise established financial facts are immutable.
- Corrections use explicit reversal, adjustment, amendment, return, unapplication, replacement, or compensation flows.
- Retriable state-changing commands require scoped idempotency.
- Integration events use a transactional outbox so database state and publication intent are durable together.
- Material state changes require authorization and audit evidence.
- Accounting ownership stays inside the bounded context that owns the fact.

The [DDD baseline](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_domain_model_ddd.md) is authoritative for domain language and invariants.
