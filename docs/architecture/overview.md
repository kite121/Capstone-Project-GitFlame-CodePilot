# Architecture Overview

GitFlame CodePilot is designed as an external integration service.

```text
GitFlame -> Backend -> ML Service
        -> Database
        -> Git workflow payloads
```

The team does not require direct access to the internal GitFlame codebase for Sprint 1.

