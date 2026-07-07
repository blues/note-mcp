# Arduino Notecard Connectivity

This document covers how to tune when and how the Notecard connects to Notehub, how to force a sync, and how to receive inbound data. It complements the `best_practices` document — apply the Notecard integration patterns there first.

## Choosing a Sync Mode

The `hub.set` request controls how the Notecard connects to Notehub. Set it once during initialisation. The `mode`, `outbound`, and `inbound` arguments are **persistent** — they are retained across restarts and across later `hub.set` requests that omit them.

- `continuous` — the Notecard keeps a live session with Notehub. Lowest latency, highest power draw. Use for mains-powered devices or during development for easy debugging.
- `periodic` — the Notecard connects on a schedule (see `outbound`/`inbound`). This is the correct default for battery-powered devices.
- `minimum` — the Notecard only connects when explicitly told to via `hub.sync`. Lowest power.
- `off` — no automatic connections.

```cpp
J *req = notecard.newRequest("hub.set");
JAddStringToObject(req, "product", myProductID);
JAddStringToObject(req, "mode", "periodic");
JAddNumberToObject(req, "outbound", 60); // upload pending Notes at most every 60 minutes
JAddNumberToObject(req, "inbound", 240); // check for inbound Notes at most every 240 minutes
notecard.sendRequestWithRetry(req, 5);
```

ALWAYS:

- start development in `continuous` mode for easy debugging, and leave a comment telling the user to switch to `periodic` for their production power profile.
- use `sendRequestWithRetry()` for the first `hub.set` on cold boot to absorb the hardware race condition (see the `index` document).

## Understanding `outbound` and `inbound`

- `outbound` is the maximum time (in minutes) the Notecard will hold pending outbound Notes before connecting to upload them.
- `inbound` is how often (in minutes) the Notecard connects to check for data queued for the device in Notehub.

Set these to the largest values your application can tolerate — larger intervals mean fewer connections and dramatically lower power and cellular data usage. Avoid very short intervals (< 5 minutes), which effectively force a continuous connection and can conflict with GPS/GNSS usage on the same Notecard.

## Forcing an Immediate Sync

Two mechanisms let you override the schedule when data must move now:

- Add `"sync": true` to a `note.add` request to have the Notecard connect and upload immediately after enqueuing that Note. Use sparingly on battery-powered devices — each sync costs power.

```cpp
J *req = notecard.newRequest("note.add");
JAddStringToObject(req, "file", "readings.qo");
JAddBoolToObject(req, "sync", true); // upload this Note right away
J *body = JAddObjectToObject(req, "body");
if (body != NULL) {
    JAddNumberToObject(body, "temperature", temperature);
}
notecard.sendRequest(req);
```

- Issue a `hub.sync` request to manually trigger a full sync of pending inbound and outbound Notefiles:

```cpp
J *req = notecard.newRequest("hub.sync");
notecard.sendRequest(req);
```

  `hub.sync` accepts `"out": true` to sync only outbound Notefiles, `"in": true` for inbound only, and `"allow": true` to release the Notecard from a connectivity penalty box.

## Checking Connection and Sync Status

Use these read requests (never write configuration in a status check) to observe connectivity:

- `hub.sync.status` — reports the state of the current or most recent sync attempt, including whether a sync is in progress and any error.
- `hub.status` — reports the current connection state to Notehub.
- `card.wireless` — reports cellular signal strength and modem state, useful when diagnosing poor connectivity.

Check `notecard.responseError(rsp)` on each of these before reading fields, and always `deleteResponse(rsp)` (see the `best_practices` document).

## Receiving Inbound Data

To act on data or commands sent from Notehub to the device, poll an inbound (`.qi`) Notefile. Do this from the main loop on a non-blocking timer, not with a blocking `delay()`.

Retrieve one Note at a time with `note.get`, deleting it as it is read:

```cpp
J *req = notecard.newRequest("note.get");
JAddStringToObject(req, "file", "commands.qi");
JAddBoolToObject(req, "delete", true); // remove the Note once retrieved

J *rsp = notecard.requestAndResponse(req);
if (notecard.responseError(rsp)) {
    // No notes available (returns a {note-noexist} error) or the request failed.
} else {
    J *body = JGetObject(rsp, "body");
    if (body != NULL) {
        const char *command = JGetString(body, "command");
        // ... act on the command ...
    }
}
notecard.deleteResponse(rsp);
```

To process several queued Notes at once, use `note.changes` (which returns all pending Notes in the Notefile) instead of `note.get`. A common robust pattern is to send an acknowledgment Note back to a `.qo` Notefile after acting on a command, so the cloud application knows the action was applied.

In `periodic` or `minimum` mode, inbound Notes only arrive on the `inbound` schedule or after a `hub.sync`. If your application needs prompt commands, either shorten `inbound`, call `hub.sync` when appropriate, or use inbound signals for low-latency delivery.

## Resilience When Offline

The Notecard's store-and-forward design means outbound Notes are queued locally and delivered when connectivity returns — the host does not need to manage retries for outbound data. For host-side robustness:

- do not block the loop waiting for a connection; enqueue Notes and let the Notecard sync on its schedule.
- consider the reserved environment variables that let the Notecard recover a stalled system: `_restart_host_no_activity_hours` restarts the connected host after no host requests have been received for the given number of hours, and `_restart_host_every_hours` restarts the host on a fixed interval. To restart the Notecard itself when it cannot reach Notehub, use `_restart_no_activity_hours`.

## Further Reading

Queries about connectivity, sync behaviour, or Notehub specifics should be made to the `docs_search` or `docs_search_expert` tool. Use the `api_docs` tool for the full argument list of `hub.set`, `hub.sync`, `note.get`, and `note.changes`.
