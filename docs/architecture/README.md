# Architecture Diagrams

This folder contains two standalone architecture images for the Cross-Team Knowledge Relay project.

## High-Level Design (HLD)

![Cross-Team Knowledge Relay HLD](./hld-architecture.svg)

The HLD diagram shows the full system boundary: users, Next.js frontend, API gateway, Go backend services, Neo4j graph matching, PostgreSQL transactional storage, Redis, event streaming, notification delivery, external scheduling, deployment, security, and observability.

## Low-Level Design (LLD)

![Cross-Team Knowledge Relay LLD](./lld-architecture.svg)

The LLD diagram expands the bounty creation and matching flow: frontend payload fields, HTTP contracts, Go request decoding, validation, skill normalization, bounty building, matching/scoring logic, Neo4j graph schema and Cypher query shape, persistence, escrow state transitions, asynchronous worker execution, notification payloads, reliability controls, and endpoint error paths.
