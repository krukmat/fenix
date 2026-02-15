# Requirements Management (Doorstop)

## Prerequisites

`./.venv/bin/doorstop`

## Adding a New Requirement

1. Create FR item (`doorstop add FR`) and edit YAML.
2. Add tests with `// Traces: FR-NNN`.
3. Create TST item (`doorstop add TST`) + link (`doorstop link TST_NNN FR_NNN`) and set `ref:`.
4. Implement feature.
5. Update `docs/openapi.yaml` if API changes.
6. Validate:
   - `./.venv/bin/doorstop`
   - `make trace-check`
   - `make test`
   - `make contract-test`

## Publishing Reports

`./.venv/bin/doorstop publish all ./docs/trace-report`
