# Getting Started

The Arduino Library, note-arduino, is a library for interacting with the Notecard.
It is a wrapper around the note-c library and provides a native Arduino interface for interacting with the Notecard.

This library is recommended for Arduino-based devices, in particular those that use the Arduino IDE/CLI for project management, building, and flashing.

## Installing the note-arduino library

If required, the user may wish to install the note-arduino library using the Arduino CLI.

### Arduino CLI

```bash
arduino-cli lib install "Blues Wireless Notecard"
```

## Code Layout

When either creating a new project or retrofitting existing projects to use the Notecard, ensure that the following structures are present:

- ALL Notecard code should be in a separate library file from the main sketch (VERY IMPORTANT).
- Limit modifications to the user's Arduino code; prefer to add new functions to the library file.
- A 'init' function that initializes the Notecard, along with `hub.set` commands to configure the Notecard.
  - Before optimising the code, set the `mode` to `continuous` for easy debugging. Use a comment to indicate that the user should change this to `periodic` to fit their application.
- The first `sendRequest()` call should instead be `sendRequestWithRetry()` to ensure that the request is retried if it fails. This is to address a potential race condition on cold boot.
- Once the initial pass has been made, go to [power management](#power-management) to ensure that the Notecard is power efficient.

## Design Patterns

Work through the following design patterns to ensure that target projects are easy to maintain and extend.

ALWAYS:

- use templates for notes and ensure that the data types are correct. ALWAYS use the `firmware_best_practices` tool with the `templates` document type for more information.
- use the I2C interface for Notecard communication, unless instructed otherwise by the user.
- use Blues Expert tools to check and validate Notecard requests and responses.
- generate a header comment in bother the `.c` and the `.h` files that contains the following information:

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

## Power Management

Implement the initial changes following this guide and then ask the user if they would like to optimise the code for power management.

- use the `firmware_best_practices` tool with the `best_practices` document type.

## Further Reading

Queries about note-arduino specifics should be made to the `docs_search` or `docs_search_expert` tool.
Additionally, the following resources may be useful:

- [note-arduino GitHub Repository](https://github.com/blues/note-arduino)
