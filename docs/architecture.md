# Chat API Architecture

## WebSocket Architecture

``` text
WebSocket Client
      |
  WebSocket Handler
      |
  JWT verification + Origin check
      |
  WebSocket Upgrade
      |
  Client.SetIdentity()
      |
  Hub.Register()
      |
+----------------------+
|                      |
ReadPump            WritePump
(sync)              (goroutine)
|                       |
|                       |
Hub.Broadcast()      client.send
Hub.Join()
|
Room-based routing
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

-   HTTP/WebSocket request parsing
-   Transport-level validation
-   JWT verification before WebSocket upgrade
-   WebSocket origin checking
-   HTTP responses
-   WebSocket upgrade

### Service

-   User registration and login
-   Business validation
-   Bcrypt password hashing/verification
-   JWT generation and verification

### Repository

-   Database operations
-   Parameterized SQL queries
-   User persistence

### Hub

-   Sole ownership of the connected-client registry
-   Register/unregister clients
-   Broadcast messages
-   Room membership
-   Room-based routing
-   Channel-based coordination

### Client

-   Authenticated WebSocket session
-   WebSocket connection ownership
-   User identity and room state
-   ReadPump (synchronous, inbound messages)
-   WritePump (goroutine, outbound messages)
-   Buffered send channel

## Graceful Shutdown

``` text
SIGINT / SIGTERM
     |
HTTP Server Shutdown
     |
Hub.Stop()
     |
Close WebSocket connections
     |
Close PostgreSQL connection pool
```

Shutdown order ensures no new work enters the system before in-flight
work completes. The HTTP server stops first so no new connections can
arrive. The Hub then closes all WebSocket connections. The database
connection pool closes last.

## Important Principle

The Hub should coordinate real-time connections. It should not become a
place where database queries, authentication logic, or unrelated
business rules are placed.
