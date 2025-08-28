# Getting Started

The Zephyr SDK, note-zephyr, is a Zephyr West Module for interacting with the Notecard.
It is a wrapper around the note-c library and provides a native Zephyr interface for interacting with the Notecard, including support for device tree bindings and KConfig options.

This SDK is recommended for Zephyr-based devices, in particular those that use the West utility for project management, building, and flashing.

## Installing the note-zephyr SDK

Add the following to your `west.yml` file:

```yaml
manifest:
  projects:
    - name: note-zephyr
        path: modules/note-zephyr
        revision: main
        submodules: true
        url: https://github.com/blues/note-zephyr
```

Then, run the following command to update your project:

```bash
west update
```

## Further Reading

Queries about note-zephyr specifics should be made to the `docs_search` or `docs_search_expert` tool.
Additionally, the following resources may be useful:

- [note-zephyr GitHub Repository](https://github.com/blues/note-zephyr)
