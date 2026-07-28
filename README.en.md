# DevOps Interview Project

[中文](README.md) | English

> Suggested effort: 2–3 hours | AI tools are allowed

## Background

This repository contains a simple Go REST API for creating, reading, updating, and deleting tasks. It also exposes `/healthz` and `/metrics`. The current repository is only a locally runnable starting point.

Your goal is to move it toward an engineering solution that is **buildable, deliverable, observable, and diagnosable** within the time you choose to invest. We care not only about whether the final files exist, but also about how you identify problems, prioritize work, validate results, and handle unfinished areas.

## Working Guidelines

- The suggested effort is **2–3 hours**, not a hard limit. You may spend more time; record your actual time, unfinished work, and next steps honestly in `deploy/NOTES.md`.
- The assignment does not provide every detail of a production environment. When information is missing, record the key assumptions you need and continue; you do not need to wait for the interviewer to provide a single correct answer.
- Tasks 1–5 are parallel evaluation dimensions, and you may choose the order. Steps 3A–3C are only the internal validation sequence for the observability task.
- You may use ChatGPT, Claude, Copilot, Cursor, or other tools, but you must be able to explain, run, and modify every part of your submission.
- You do not need to purchase cloud resources. The deployment target may be a local or remote environment that you can demonstrate.
- No specific tool or implementation is required. Any approach is acceptable if it satisfies the behavioral constraints below and you can explain your choices.
- The interview will use your submission directly for validation and live changes. It will not focus on memorizing tool-specific knowledge.

## Key Files

```text
.
├── main.go
├── handler.go
├── handler_test.go
├── store.go
├── store_test.go
├── Dockerfile
├── Makefile
├── docker-compose.yml
├── .github/workflows/ci.yml
├── monitoring/prometheus.yml
├── deploy/NOTES.md
└── deploy/COST.md
```

Local baseline:

```bash
go test -race -count=1 ./...
go run .
```

API:

```text
GET    /healthz
GET    /metrics
GET    /tasks
POST   /tasks
GET    /tasks/{id}
PUT    /tasks/{id}
DELETE /tasks/{id}
```

## Task 1: Containerization

Package the application as a runnable container image.

**Acceptance outcomes:**

- `docker build -t task-api .` succeeds;
- after the container starts, `/healthz` returns `{"status":"ok"}`;
- Docker actually reports the container as `healthy`; merely having a `HEALTHCHECK` instruction in the Dockerfile is not sufficient;
- the final image is smaller than 15 MiB, measured using the byte value returned by `docker image inspect task-api --format '{{.Size}}'`;
- the application process does not run as root.

The current Dockerfile is only a starting point. Inspect the resulting artifact and its runtime behavior yourself.

## Task 2: CI/CD

Implement a pipeline that can validate, publish, and deploy this project. You may use GitHub Actions or another platform.

**Behavioral constraints:**

- pushes to `main` and pull/merge requests targeting `main` trigger the necessary automated validation;
- the overall delivery design covers static analysis, tests, compilation, container image build, image publication, and deployment, but not every event needs to execute every stage;
- you decide which external side effects are allowed for pull/merge requests, pushes to `main`, manually approved events, or equivalent events on your chosen platform;
- published images can be traced to a specific commit;
- credentials do not appear in plaintext in the repository or logs;
- deployment is not an `echo`, comment, or pseudocode: it must include actual commands that create or update a runtime environment. The target may be local, temporary, or remote, and it does not need to expose a long-lived public URL;
- the actual execution time target for jobs on the automated validation path is under 10 minutes, excluding runner queueing, manual approval, and waiting for external environments.

If publication or deployment cannot run online because of public-fork behavior, credentials, or external-environment limitations, keep an executable implementation with safe conditions, perform local or equivalent end-to-end validation, provide the resulting evidence, and document the validation boundary in `deploy/NOTES.md`.

## Task 3: Observability

### 3A. Run the monitoring stack

Complete `docker-compose.yml` and `monitoring/` so that the application, Prometheus, and Grafana can run together.

**Acceptance outcomes:**

- the Prometheus target is shown as `UP`;
- Grafana can query Prometheus data;
- the repository contains a Dashboard that can be imported or provisioned automatically;
- after starting from a clean environment and generating your documented test traffic, the Dashboard shows real data rather than remaining empty because of a datasource, query, or time-range problem.

### 3B. Make the metrics useful for on-call decisions

The existing metrics are not sufficient to determine whether the API is healthy. Modify the application and Dashboard so that an on-call engineer can answer:

- What is the current request volume and failure behavior?
- What are the P50, P95, and P99 latencies for business requests?
- What is the current task state?

You decide the metric types, labels, statistical boundaries, and sampling approach. Keep `/metrics` as the Prometheus scrape endpoint, and add the necessary tests for new application code.

### 3C. Validate the observations

Use the existing API to generate the business traffic and state transitions that you consider sufficient to validate the Dashboard. You decide the method and operation mix.

Before running the experiment, write down what you expect each relevant panel to do. Afterward:

1. compare the Dashboard and raw queries with your expectations, and record whether the actual changes match;
2. select one signal for further confirmation and explain why it deserves attention;
3. if you observe an anomaly, missing data, or behavior you cannot explain, use additional queries, commands, or experiments to distinguish service behavior, observability implementation, and the experiment itself;
4. use the evidence to decide whether to change the implementation, and record the conclusion and next step in `deploy/NOTES.md`;
5. if you make a change, rerun the relevant steps to validate your conclusion.

We do not expect a large number of panels. A small number of trustworthy signals that help you discover and explain issues is more valuable.

## Task 4: Submit a Decision Record

Complete `deploy/NOTES.md`. It is not a generic DevOps questionnaire; it is an evidence index for this implementation, including:

- the key assumptions on which your implementation depends;
- the actual delivery path and rollback unit;
- one validation or investigation that you actually performed;
- two engineering trade-offs you deliberately made, and the conditions that would cause you to change those decisions;
- actual time spent, unfinished work, and next steps;
- if you used AI, one concrete example of AI output that you reviewed, modified, or rejected.

Reference actual files, jobs, commands, or runtime results. Avoid generic statements such as “in production, we should…”. Keep it concise; for the English version, aim for no more than roughly 1,000 words, equivalent to the Chinese version’s 1,500-character guideline.

## Task 5: Cost Scenario Analysis

> This is a **scenario analysis** task. It requires no code and no cloud purchases. Write your analysis in `deploy/COST.md` (aim for no more than ~600 words). **If you can obtain the data, give concrete numbers and queries; if you cannot, state your key assumptions, what data you would use to verify them, and how your conclusion would change if they do not hold.** We care about both your conclusion and your reasoning.

Imagine the `task-api` from Tasks 1–4 as one part of a larger production system. The system also has a high-write user-event table `user_event`: it is written to **DynamoDB**, its data is backed up/synced to an **S3** data-lake bucket, downstream services on **EKS** consume the events, and outbound traffic goes through **ELB**. The app recently shipped a major revision, and the client request logic changed with it. Afterward:

- the monthly **AWS bill rose about 30% month over month** (illustrative: ~$40k → ~$52k);
- over the same period, **business volume (DAU / core request volume) grew only about 5%**;
- the increase is not confined to one service; it is **spread across DynamoDB, S3, EKS compute, and ELB egress**;
- you have access to: Cost Explorer, an exportable CUR (Cost and Usage Report), roughly 60% of resources tagged with `team` / `env`, CloudWatch metrics, and the service metrics you built in Task 3;
- Finance and your leader want two answers: **why did it rise**, and **can we bring it down next month, and how**.

**In `deploy/COST.md`, provide:**

- **Baseline judgment:** which data do you look at first to decide whether this increase is "normal"? How do you separate "usage growth" from "efficiency loss / waste"? What unit do you measure by (for example, unit cost per event / request / DAU)?
- **Attribution approach:** give one "observe → hypothesize → verify with which data" chain that breaks the 30% increase down to specific sources. Cover at least: which dimensions you split by first (service / usage type / account / region / tag) and how you attribute the untagged ~40% of resources; when the increase is spread across services, how you decide whether they share a **common root cause**; and the confounders you rule out at each step (billing-period day count, RI/SP amortization, one-off charges).
- **Governance and trade-offs:** pick one constraint and give a concrete first step, an alternative, and what you give up — (1) you want to use commitment discounts (RI / Savings Plans) or reserved capacity to cut cost, but this event table is **scheduled to migrate to a different store next quarter**; (2) the root cause is the post-revision request/write logic, and **reducing it at the source needs the APP team, who are fully booked this quarter**; (3) last month you predicted the cost would drop, but this month it hit a **new high** (your point optimizations were outweighed by growth at the source) — how do you **manage expectations honestly** with your leader.
- **Evidence from experience (optional but encouraged):** if you have done a real cost optimization, describe the before/after result in one or two sentences and how you confirmed the improvement actually came from your change.

**Behavioral constraints:**

- Conclusions must land on "which data, verified how"; avoid a generic checklist such as "shut down idle resources / buy RIs / add a lifecycle policy" — for the same action, explain **why you judge it applies to this scenario**;
- Distinguish a "one-off point optimization" from "root-cause governance / a durable mechanism," and explain how you would prevent the cost from creeping back.

## Core Acceptance Checklist

- [ ] The original Go tests pass, with no data races;
- [ ] the image runs, becomes healthy in practice, is smaller than 15 MiB, and runs the application as a non-root user;
- [ ] CI performs actions appropriate to the risk level of pull/merge requests and pushes to `main`;
- [ ] image publication and deployment are real and traceable;
- [ ] the Prometheus target is `UP`, and the Grafana Dashboard contains real data;
- [ ] metrics can answer questions about traffic, errors, latency percentiles, and task state;
- [ ] the Dashboard has been validated with actual business traffic, actual results have been compared with expectations, and at least one signal has been confirmed further;
- [ ] `deploy/NOTES.md` references real evidence from this submission;
- [ ] `deploy/COST.md` attributes the cost increase to evidence or explicit assumptions, states a governance trade-off, and separates point optimizations from root-cause governance.

## Evaluation Focus

| Dimension | What we look for |
|---|---|
| Debugging and validation | Whether you gather evidence first, form hypotheses, control variables, and validate again |
| Engineering judgment | Whether you identify conflicting constraints, make trade-offs, and explain remaining risks |
| Correctness | Whether runtime behavior matches your claims and remains correct when boundaries change |
| Delivery safety | Whether permissions, credentials, publication conditions, and artifact traceability are reasonable |
| Observability | Whether metrics and the Dashboard actually support decisions and troubleshooting |
| Prioritization | Whether you prioritize the highest-value closed loops within the time you actually invest |
| Cost judgment | Whether you show unit-cost awareness, attribute the increase to evidence, separate point fixes from root-cause governance, and make an actionable trade-off under constraints |

## Optional Bonus

After completing the core work, choose at most **one** item to continue implementing. Not completing a bonus item does not prevent a strong evaluation of the core work.

- container image scanning and a policy for handling the result;
- graceful shutdown with repeatable validation;
- a readiness mechanism with demonstrated state transitions;
- an alert that can be triggered and shown to recover;
- artifact promotion from staging to production;
- a queryable SLI/SLO for this service.

What you choose is less important than why it deserves the remaining time more than the alternatives, and whether you actually validated it.

## FAQ

**May I use AI?** Yes. You do not need to submit the full conversation history, but you are responsible for the final result. During the interview, you may be asked to explain or modify AI-generated parts.

**Must I deploy to a public environment?** No. The deployment target must be executable and demonstrable, but it may be local. Do not purchase resources for this assignment.

**Must I stop after 2–3 hours?** No. This is suggested effort, not a hard limit. Regardless of how long you spend, record what you validated, why anything remains unfinished, and what you would do next. Good prioritization and validation matter more than accumulating files.
