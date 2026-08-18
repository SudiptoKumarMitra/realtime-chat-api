# Chat API Architecture

## Initial WebSocket Architecture

``` text
                    WebSocket Clients
                 /        |        \
                A         B         C
                 \        |        /
                       WebSocket
                          |
                    WebSocket Handler
                          |
                         Hub
                          |
              Connected Client Registry
```

## HTTP Architecture

``` text
HTTP Request
     |
   Handler
     |
   Service
     |
 Repository
     |
 PostgreSQL
```

## Responsibilities

### Handler

-   Parse HTTP/WebSocket requests
-   Validate transport-level input
-   Return HTTP responses
-   Establish WebSocket connections

### Service

-   Business rules
-   Conversation/message operations
-   Authorization decisions

### Repository

-   Database operations
-   Queries
-   Persistence

### Hub

-   Track active WebSocket clients
-   Register clients
-   Unregister clients
-   Broadcast messages
-   Coordinate real-time delivery

### Client

-   Represent one connected WebSocket user/session
-   Own the WebSocket connection
-   Receive/send data through controlled channels

## Important Principle

The Hub should coordinate real-time connections. It should not become a
place where database queries, authentication logic, or unrelated
business rules are placed.
