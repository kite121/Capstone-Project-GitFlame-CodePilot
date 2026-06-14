# GitFlame CodePilot

GitFlame CodePilot is an external AI integration service for GitFlame. It is designed to receive repository events and context from GitFlame, analyze repository configuration, generate Markdown implementation plans for issues, prepare generated code file payloads after approval, and return repository recommendations.

## Repository Structure

```text
backend/       Main API service and GitFlame integration contracts
frontend/      Demo UI for product flows and screenshots
ml_service/    Open-source model integration and mock AI endpoints
docs/          Report sections, architecture notes, schemas, and diagrams
infra/         Deployment and infrastructure notes
```

## Current Sprint Scope

Sprint 1 focuses on project setup, architecture, report materials, initial API contracts, `.yml` configuration draft, and runnable skeleton services.

## Quick Start

Backend:

```bash
cd backend
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000
```

ML service:

```bash
cd ml_service
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8001
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

## Health Checks

```text
Backend:    GET http://localhost:8000/health
ML service: GET http://localhost:8001/health
```

