# TDD: Goal Verifier Dashboard Read-Model Checks Alias

## Tests

1. Add a focused verifier-script test that asserts
   `dashboard_read_models = $dashboardReadModelChecks` is present under the
   top-level `checks` object.
2. Rerun the existing focused verifier tests.
3. Rerun `scripts/verify-go-migration-goal.ps1` and inspect the artifact to
   confirm the aliased path is materialized.
