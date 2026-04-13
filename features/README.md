# BDD Features

This directory contains the executable BDD behavior catalog for the project.

Current conventions:

- one or more `.feature` files per `UC`
- all feature files are written in English
- every scenario must include `@UC-*`, `@FR-*`, `@TST-*`, and one `@stack-*` tag
- `@stack-go` is the canonical backend/contract runner
- `@stack-mobile` covers declared mobile behavior; execution remains blocked in CI until the Android-backed mobile BDD runner is available
- `@deferred` marks roadmap or advanced coverage intentionally excluded from the default Go suite
- business UCs should be introduced during Wave 3 before AGENT_SPEC behavior families

Current wedge-critical coverage:

- `uc-s1-sales-copilot.feature`
- `uc-c1-support-agent.feature`
- `uc-g1-governance.feature`
- `uc-s1-sales-copilot-mobile-smoke.feature`
- `uc-s2-prospecting-agent-mobile.feature`
- `uc-k1-kb-agent-mobile.feature`
- `uc-d1-data-insights-agent-mobile.feature`
- `uc-s3-deal-risk-agent-mobile.feature`
