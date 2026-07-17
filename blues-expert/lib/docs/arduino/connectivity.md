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

Check `notecard.responseError(rsp)` on each of these before reading fields, and always `deleteResponse(rsp)` (see the `best_practices` document). To watch a sync progress live while debugging, use the `debugSyncStatus()` helper described in the `debugging` document.

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

## Interrupt-Driven Inbound With the ATTN Pin

Polling in a fast loop wastes power and host cycles. When the host can spare a GPIO, prefer letting the Notecard interrupt the host only when an inbound Notefile actually changes. Wire the Notecard's `ATTN` pin to a host GPIO and use the `card.attn` request to arm a file-watch interrupt. This is the interrupt-driven pattern the General Embedded Firmware Best Practices section (in the `best_practices` document) refers to: the ISR does nothing but set a `volatile` flag, and all real work happens back in `loop()`.

Arm the interrupt in `setup()` to fire when a specific inbound Notefile is modified. `mode` is a comma-separated list; `arm,files` clears any prior event, watches the Notefiles in `files`, and pulls `ATTN` low until one of them changes (or `seconds` elapses, if non-zero):

```cpp
#define ATTN_INPUT_PIN 5 // any interrupt-capable GPIO on the host
#define INBOUND_QUEUE_NOTEFILE "my-inbound.qi"

// Set by the ISR, read by loop(). MUST be volatile — it is shared with an
// interrupt context (see the best_practices document).
static volatile bool attnInterruptOccurred = false;

void attnISR() {
    attnInterruptOccurred = true; // do nothing else in the ISR
}

void armAttn() {
    attnInterruptOccurred = false;
    J *req = notecard.newRequest("card.attn");
    JAddStringToObject(req, "mode", "arm,files");
    const char *files[] = { INBOUND_QUEUE_NOTEFILE };
    JAddItemToObject(req, "files", JCreateStringArray(files, 1));
    JAddNumberToObject(req, "seconds", 120); // also wake at least every 2 min; 0 = no timeout
    notecard.sendRequest(req);
}

void setup() {
    // ... notecard.begin() and hub.set as usual ...
    pinMode(ATTN_INPUT_PIN, INPUT);
    attachInterrupt(digitalPinToInterrupt(ATTN_INPUT_PIN), attnISR, RISING);
    armAttn();
}
```

In `loop()`, do nothing until the flag is set, then drain the inbound Notefile and re-arm:

```cpp
void loop() {
    if (!attnInterruptOccurred) {
        return; // or sleep / do other non-blocking work
    }
    armAttn(); // re-arm before processing so no change is missed

    // Drain all pending inbound Notes.
    while (true) {
        J *req = notecard.newRequest("note.get");
        JAddStringToObject(req, "file", INBOUND_QUEUE_NOTEFILE);
        JAddBoolToObject(req, "delete", true);
        J *rsp = notecard.requestAndResponse(req);
        // responseError is expected (and ends the loop) once the queue is
        // empty: a {note-noexist} or {file-noexist} error just means "no more".
        if (notecard.responseError(rsp)) {
            notecard.deleteResponse(rsp);
            break;
        }
        J *body = JGetObject(rsp, "body");
        if (body != NULL) {
            // ... act on the command ...
        }
        notecard.deleteResponse(rsp);
    }
}
```

Notes:

- Re-arming with `arm,files` is non-idempotent (it briefly pulls `ATTN` high, then low again). To re-arm using the values from the initial arm without restating them, use `mode: "rearm"` instead.
- To clear all ATTN monitors, send `mode: "disarm,-all"`.
- `card.attn` requires a physical wire from the Notecard `ATTN` pin to the chosen host GPIO. `ATTN` can watch many other events too (motion, location fixes, environment-variable changes, USB power); use `docs_search` or the `api_docs` tool for the full `mode` list.

## Resilience When Offline

The Notecard's store-and-forward design means outbound Notes are queued locally and delivered when connectivity returns — the host does not need to manage retries for outbound data. For host-side robustness:

- do not block the loop waiting for a connection; enqueue Notes and let the Notecard sync on its schedule.
- consider the reserved environment variables that let the Notecard recover a stalled system: `_restart_host_no_activity_hours` restarts the connected host after no host requests have been received for the given number of hours, and `_restart_host_every_hours` restarts the host on a fixed interval. To restart the Notecard itself when it cannot reach Notehub, use `_restart_no_activity_hours`.

## Further Reading

Queries about connectivity, sync behaviour, or Notehub specifics should be made to the `docs_search` tool. Use the `api_docs` tool for the full argument list of `hub.set`, `hub.sync`, `note.get`, and `note.changes`.
