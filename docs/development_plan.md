# Development Plan

## Phase 1 --- WebSocket Fundamentals

-   Minimal WebSocket server
-   Echo message
-   Understand connection lifecycle

## Phase 2 --- Concurrency + Hub

-   Client structure
-   Hub structure
-   Register
-   Unregister
-   Broadcast
-   Select loop
-   Multiple clients

## Phase 3 --- Robust WebSocket Handling

-   Read pump
-   Write pump
-   Send channel
-   Ping/pong
-   Deadlines
-   Graceful disconnect
-   Context cancellation

## Phase 4 --- Chat Features

-   User identity
-   One-to-one conversations
-   Chat rooms
-   Message types
-   Online/offline presence

## Phase 5 --- Authentication

-   Register
-   Login
-   Password hashing
-   JWT
-   WebSocket authentication

## Phase 6 --- PostgreSQL

-   Schema
-   Migrations
-   Users
-   Conversations
-   Members
-   Messages
-   Repository layer

## Phase 7 --- Production Concerns

-   Validation
-   Error handling
-   Logging
-   Configuration
-   Docker
-   Tests
-   Race detector
-   Graceful shutdown

## Phase 8 --- Scaling

Only after the single-server version is solid:

-   Redis
-   Pub/Sub
-   Multiple WebSocket servers
-   Load balancing
-   Presence synchronization
