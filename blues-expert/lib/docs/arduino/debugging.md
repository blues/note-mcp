# Arduino Notecard Debugging

Getting visibility into what the Notecard and host are doing is the fastest way to find problems. Start every new Arduino project with debug output enabled, confirm the behaviour is correct, and only then disable it (see the `best_practices` document).

## Enabling Notecard Debug Output

`note-arduino` can print every JSON request and response, plus the Notecard's own internal log messages, to a serial stream. Wire this up in `setup()`:

```cpp
#include <Notecard.h>

#define usbSerial Serial

Notecard notecard;

void setup() {
    usbSerial.begin(115200);
    // Wait up to 3 seconds for the serial monitor to attach (non-blocking bound).
    const size_t usb_timeout_ms = 3000;
    for (const size_t start_ms = millis(); !usbSerial && (millis() - start_ms) < usb_timeout_ms;)
        ;

    // Route Notecard debug output (and JSON traffic) to the serial monitor.
    // Guard low-memory hosts, where internal logs are compiled out.
#ifndef NOTE_C_LOW_MEM
    notecard.setDebugOutputStream(usbSerial);
#endif

    notecard.begin(); // I2C by default
}
```

With `setDebugOutputStream()` set, opening the Arduino serial monitor at 115200 baud shows the full request/response traffic to and from the Notecard, which is the single most useful debugging tool for a Notecard project.

ALWAYS:

- make the debug stream easy for the user to disable (a single `#define` or a `#ifdef` guard), since it must be removed for low-power production firmware.
- guard `setDebugOutputStream()` with `#ifndef NOTE_C_LOW_MEM` so the sketch still builds on low-memory hosts.

## Logging From Your Own Code

Use the Notecard's debug logging helpers so your application logs travel on the same stream as the Notecard's:

```cpp
notecard.logDebug("Sensor initialised\n");                 // fixed string
notecard.logDebugf("temperature=%.2f C\n", temperature);   // printf-style formatting
```

Prefer `logDebugf()` when you need formatted values — unlike Arduino's `Serial.println()`, it accepts printf-style format specifiers. These helpers are no-ops when no debug stream is configured, so they are safe to leave in place.

## Watching a Sync Complete

When debugging connectivity, it is useful to watch a sync progress in real time rather than polling `hub.sync.status` by hand. `note-arduino` provides a helper for exactly this:

```cpp
// Poll the Notecard for sync status roughly every 1000 ms and print every
// log level (-1 = all) to the debug stream until the sync settles.
J *req = notecard.newRequest("hub.sync");
notecard.sendRequest(req);

while (notecard.debugSyncStatus(1000, -1)) {
    // debugSyncStatus() returns true while there is pending sync activity to
    // report, and prints each status line to the configured debug stream.
}
```

`debugSyncStatus(pollFrequencyMs, maxLevel)` requires a debug output stream to have been set with `setDebugOutputStream()`. Use it during development only; remove it from production firmware since it blocks the loop while polling.

## Notecard Trace Mode

For problems that the request/response traffic alone does not explain (connectivity, GPS acquisition, sync timing), enable Notecard **trace mode**. The Notecard then streams a detailed internal log of its activity to the debug output.

```cpp
J *req = notecard.newRequest("card.trace");
JAddStringToObject(req, "mode", "on"); // use "off" to disable
notecard.sendRequest(req);
```

Trace mode is verbose and increases power consumption, so enable it only while actively debugging and turn it `off` (or remove it) before shipping.

## Debugging Over the STLinkV3

On boards where the USB serial port is not available, you can route debug output over the STLinkV3 virtual COM port:

```cpp
HardwareSerial SerialVCP(PIN_VCP_RX, PIN_VCP_TX);
#define debugSerial SerialVCP
```

Then pass `debugSerial` to `notecard.setDebugOutputStream(debugSerial)`.

## Monitoring the Notecard Directly

You do not always need the host to see what the Notecard is doing:

- **In-browser terminal** — connect the Notecard over USB and use the Blues in-browser terminal to issue requests and read responses directly. This is invaluable for isolating whether a problem is in the Notecard configuration or in the host firmware.
- **AUX serial / FTDI debug cable** — the Notecard's `AUX` pins can mirror request/response traffic. Configure them with `card.aux.serial` using `"mode": "req"` and connect an FTDI cable to watch the traffic on a second serial port. This is useful when the host's only serial port is already in use.

## Common Debugging Checklist

When a Notecard project is not behaving as expected:

1. Confirm `setDebugOutputStream()` is set and the serial monitor is open at 115200 baud.
2. Check `notecard.responseError(rsp)` on every response and log failures (see the `best_practices` document).
3. Verify the `hub.set` `product`, `mode`, and interval arguments with a `hub.get` request.
4. Check connectivity and sync state with `hub.sync.status` and `card.wireless` (see the `connectivity` document).
5. If the issue is timing- or connection-related, enable `card.trace` mode.

## Further Reading

Queries about `note-arduino` debugging specifics should be made to the `docs_search` tool.
