# Getting Started

The Python SDK, note-python, is a library for interacting with the Notecard.
It is a wrapper around the Notecard API and provides a more fluent Pythonic interface for interacting with the Notecard.

This library is recommended for high-level interactions with the Notecard, such as from a Raspberry Pi or other Linux-based device.
Support for CircuitPython and MicroPython is also available.

## Installing the note-python SDK

If required, the user may wish to install the note-python library using pip.

```bash
pip install note-python
```

Or add it to `requirements.txt`:

```txt
note-python>=1.2.0
```

## Code Layout

When either creating a new project or retrofitting existing projects to use the Notecard, ensure that the following structures are present:

- ALL Notecard code should be in a separate module/file from the main application (VERY IMPORTANT).
- Limit modifications to the user's main application code; prefer to add new functions to the Notecard module.
- An 'init' or 'setup' function that initializes the Notecard connection, along with `hub.set` commands to configure the Notecard.
  - Before optimizing the code, set the `mode` to `continuous` for easy debugging. Use a comment to indicate that the user should change this to `periodic` to fit their application.
- Include proper error handling with try/except blocks for all Notecard transactions.
- Once the initial pass has been made, go to [power management](#power-management) to ensure that the Notecard is power efficient.

## Design Patterns

Work through the following design patterns to ensure that target projects are easy to maintain and extend.

ALWAYS:

- use templates for notes and ensure that the data types are correct. ALWAYS use the `firmware_best_practices` tool with the `templates` document type for more information.
- use the I2C interface for Notecard communication (default on Raspberry Pi), unless instructed otherwise by the user.
- use Blues Expert tools to check and validate Notecard requests and responses.
- generate a header docstring in the Notecard module that contains the following information:

```python
"""
<MODULE_NAME> - Module for <DESCRIPTION>

This module encapsulates all Notecard functionality for the <PROJECT_NAME>.
This is specific to your project and is NOT A GENERAL PURPOSE LIBRARY.

THIS FILE SHOULD BE EDITED AFTER GENERATION.
IT IS PROVIDED AS A STARTING POINT FOR THE USER TO EDIT AND EXTEND.
"""
```

TRY TO:

- use Python's logging module for debugging output. This should be easily configurable by the user.
- follow PEP 8 style guidelines for Python code.

NEVER:

- layer note-python calls within the user's main application code. If the Notecard is used, it should be handled by a function in the newly created Notecard module.

## Power Management

Implement the initial changes following this guide and then ask the user if they would like to optimize the code for power management.

- use the `firmware_best_practices` tool with the `best_practices` document type.

## Further Reading

Queries about note-python specifics should be made to the `docs_search` or `docs_search_expert` tool.
Additionally, the following resources may be useful:

- [note-python GitHub Repository](https://github.com/blues/note-python)
