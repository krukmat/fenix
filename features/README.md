# BDD Features

This directory contains the executable BDD behavior catalog for the project.

Current conventions:

- one or more `.feature` files per `UC`
- all feature files are written in English
- every scenario must include `@UC-*`, `@FR-*`, `@TST-*`, and one `@stack-*` tag
- business UCs should be introduced during Wave 3 before AGENT_SPEC behavior families

Initial Wave 3 coverage:

- `uc-s1-sales-copilot.feature`
- `uc-c1-support-agent.feature`
- `uc-g1-governance.feature`
