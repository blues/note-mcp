# Getting Started

The Arduino Library, note-arduino, is a library for interacting with the Notecard.
It is a wrapper around the note-c library and provides a native Arduino interface for interacting with the Notecard.

This library is recommended for Arduino-based devices, in particular those that use the Arduino IDE/CLI for project management, building, and flashing.

Use this document to orient yourself, then follow the [Recommended Workflow](#recommended-workflow) below, pulling in detailed guidance from the `firmware_best_practices` tool as each step calls for it.

## Installing the note-arduino library

If required, the user may wish to install the note-arduino library using the Arduino CLI.

### Arduino CLI

```bash
arduino-cli lib install "Blues Wireless Notecard"
```

## Before You Start — Gather Requirements

Collect the following from the user BEFORE writing any code. Do not guess these — ask if they are not already known.

- **REQUIRED: Product UID.** The `com.company.name:product` identifier that binds the device to a Notehub project. Without it, the firmware will build and run but data will never reach Notehub. If the user does not have one, guide them to claim one at [notehub.io](https://notehub.io) (see the [ProductUID guide](https://dev.blues.io/tools-and-sdks/samples/product-uid)) before proceeding.
- **Hardware.** The host MCU and carrier board. Default to the Blues Swan (Feather) + Notecarrier-F unless the user says otherwise — several patterns (e.g. power management) assume a Notecarrier-F.
- **Connection interface.** Default to I2C for the Notecard unless the user needs serial. If serial, ask which UART/port.
- **Sensors and data.** What the user wants to measure, and the specific sensor parts. Prefer I2C sensors with an Adafruit Arduino library where possible.
- **Sync cadence.** How often data must reach Notehub. This determines the `hub.set` `mode` and `outbound`/`inbound` intervals.

## Recommended Workflow

Work through these steps in order. Each step names the `document_type` to retrieve from the `firmware_best_practices` tool (pass `sdk: arduino`).

1. **Scaffold the project.** Create the sketch directory, a `README.md`, and a separate library file for all Notecard code, following the [Code Layout](#code-layout) rules below.
2. **Write the initial integration** — Notecard init (`hub.set`) plus a basic sensor-read-and-`note.add` loop. Start in `continuous` mode for easy debugging. Retrieve `best_practices` for a complete working example, Notecard response-checking, and general embedded firmware guidance. **Read this first.**
3. **Use Note templates** for every Notefile to minimise bandwidth. Retrieve `templates`. ALWAYS use templates for Notes.
4. **Add sensors.** Retrieve `sensors` for wiring and reading common I2C parts.
5. **Confirm it works with debug output.** Retrieve `debugging` for serial output, logging, `card.trace` mode, and monitoring the request/response traffic. Verify the device is producing events in Notehub before optimising anything.
6. **Tune connectivity.** Retrieve `connectivity` to switch to `periodic` mode, set the sync intervals to match the user's cadence, force syncs when needed, and receive inbound data.
7. **Optimise for power — last.** Only after the sketch is confirmed working, ask the user if they want to reduce power consumption, then retrieve `power_management`.

Throughout: validate every Notecard request with the `api_validate` tool, and use `docs_search` / `docs_search_expert` for anything not covered here.

## Code Layout

When either creating a new project or retrofitting existing projects to use the Notecard, ensure that the following structures are present:

- ALL Notecard code should be in a separate library file from the main sketch (VERY IMPORTANT).
- Limit modifications to the user's Arduino code; prefer to add new functions to the library file.
- A 'init' function that initializes the Notecard, along with `hub.set` commands to configure the Notecard.
  - Before optimising the code, set the `mode` to `continuous` for easy debugging. Use a comment to indicate that the user should change this to `periodic` to fit their application.
- The first `sendRequest()` call should instead be `sendRequestWithRetry()` to ensure that the request is retried if it fails. This is to address a potential race condition on cold boot.
- Once the initial pass has been made, optimise for power efficiency using the `power_management` document type (step 7 above).

## Design Patterns

Work through the following design patterns to ensure that target projects are easy to maintain and extend.

ALWAYS:

- use templates for notes and ensure that the data types are correct. ALWAYS use the `firmware_best_practices` tool with the `templates` document type for more information.
- use the I2C interface for Notecard communication, unless instructed otherwise by the user.
- use Blues Expert tools to check and validate Notecard requests and responses.
- generate a header comment in both the `.cpp` and the `.h` files that contains the following information:

```c
/***************************************************************************
  <LIBRARY_NAME> - Library for <DESCRIPTION>

  This library encapsulates all Notecard functionality for the <PROJECT_NAME>.
  This is specific to your project and is NOT A GENERAL PURPOSE LIBRARY.

  THIS FILE SHOULD BE EDITED AFTER GENERATION.
  IT IS PROVIDED AS A STARTING POINT FOR THE USER TO EDIT AND EXTEND.
***************************************************************************/
```

TRY TO:

- use the Serial interface for debugging, if possible. This should be easily disabled by the user, if not needed.

NEVER:

- layer note-c calls within the user's application code. If the Notecard is used, it should be handled by a function in the newly created library file.

## Further Reading

Queries about note-arduino specifics should be made to the `docs_search` or `docs_search_expert` tool.
Additionally, the following resources may be useful:

- [note-arduino GitHub Repository](https://github.com/blues/note-arduino)
