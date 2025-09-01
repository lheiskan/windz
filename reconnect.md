# SSE Reconnection Implementation Details

## Overview
The WindZ Monitor implements intelligent SSE (Server-Sent Events) reconnection with automatic disconnect when the page is hidden and smart reconnection with catch-up when the page becomes visible again.

## Client-Side Reconnection Logic

### State Management
```javascript
let globalLastId = 0;  // Track max Unix timestamp across all events (session-only)
let isPageVisible = !document.hidden;
```

### Message ID Tracking
Every SSE message from the server includes an `id` field with a Unix timestamp. The client tracks the maximum ID seen:

```javascript
function trackGlobalId(id) {
    if (!id || typeof id !== 'number') return;
    if (id > globalLastId) {
        globalLastId = id;
    }
}
```

### Page Visibility Handling
The client listens for visibility changes and manages the SSE connection accordingly:

```javascript
document.addEventListener('visibilitychange', function() {
    if (document.hidden && isPageVisible) {
        // Page hidden - disconnect SSE to save resources
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        isPageVisible = false;
        updateConnectionStatus(false);
        
    } else if (!document.hidden && !isPageVisible) {
        // Page visible - reconnect with last known ID
        connectSSE();
        isPageVisible = true;
    }
});
```

### Reconnection with Catch-up
When reconnecting, the client sends the last known ID as a query parameter:

```javascript
function connectSSE() {
    // Build URL with id parameter for catch-up
    let url = '/events';
    if (globalLastId > 0) {
        url += '?id=' + globalLastId;  // e.g., /events?id=1756761832
    }
    
    eventSource = new EventSource(url);
    // ... event handlers ...
}
```

## Server-Side Reconnection Handling

### Reconnection Detection
The server checks for reconnection in two ways:

1. **Manual Reconnection** (via query parameter) - Used when page becomes visible:
   ```go
   if idParam := r.URL.Query().Get("id"); idParam != "" {
       // Parse Unix timestamp and send catch-up data
   }
   ```

2. **Automatic Reconnection** (via Last-Event-ID header) - Used by browser on network issues:
   ```go
   if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
       // Browser automatic retry with last received ID
   }
   ```

### Catch-up Data
When a reconnection is detected, the server sends only events newer than the provided timestamp:

- **Wind Data**: Sends readings where `UpdatedAt > lastSeen`
- **Status Updates**: Sends status changes where `LastPolled > lastSeen`

## Reconnection Scenarios

### 1. Fresh Page Load
- No `globalLastId` (starts at 0)
- Server sends all current data
- Client begins tracking message IDs

### 2. Tab Switch (Hide → Show)
- **Hide**: SSE connection closed, `globalLastId` preserved
- **Show**: New SSE connection with `?id={globalLastId}`
- Server sends only events that occurred while hidden

### 3. Network Interruption
- Browser automatically reconnects
- Uses `Last-Event-ID` header with last received ID
- Server sends catch-up data from that point

### 4. Page Refresh
- `globalLastId` resets to 0 (no localStorage)
- Fresh start with all current data

## Key Design Decisions

### Why Unix Timestamps as IDs?
- Simple integer comparisons
- Natural chronological ordering
- Smaller payload than RFC3339 strings
- Direct SSE `id:` field usage

### Why No localStorage?
- Clean page refresh behavior
- No persistent state management
- Simpler implementation
- Privacy-friendly

### Why Disconnect on Hidden?
- Saves battery on mobile devices
- Reduces bandwidth usage
- Lowers server load
- No wasted resources on invisible tabs

## Example Flow

1. User opens page → SSE connects → `globalLastId = 0`
2. Events received → `globalLastId = 1756761742`
3. User switches tab → Page hidden → SSE disconnects
4. Time passes, new data arrives on server
5. User returns to tab → Page visible → SSE reconnects with `?id=1756761742`
6. Server sends all events with ID > 1756761742
7. User sees complete, up-to-date information

## Debug Logging

When running with `-debug` flag, the server logs reconnection details:

```
SSE initial connection (no prior ID)
SSE reconnect with id parameter: 1756761578 (time: 00:19:38)
SSE catch-up completed: 16 data events, 16 status events (since 00:19:38)
SSE automatic reconnect with Last-Event-ID: 1756761832 (time: 00:23:52)
```

## Benefits

- **Resource Efficient**: No streaming to hidden tabs
- **Complete Recovery**: No missed events during disconnect
- **Standards Compliant**: Uses standard SSE reconnection mechanisms
- **Session-Only**: Clean separation between browser sessions
- **Zero Configuration**: Works automatically, no user setup needed