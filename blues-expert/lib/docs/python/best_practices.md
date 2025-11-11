# Python Notecard Best Practices

When creating a new Python project with the Notecard, there are a few best practices to follow to ensure that the project is easy to maintain and extend.

## Project Structure

- Organize your project with a clear directory structure, typically with a main application file (e.g. 'app.py' or 'main.py')
- Create a 'README.md' in the project directory. This should contain a description of the project, setup instructions, and details for how to connect any sensors to the Notecard.
- Create a 'requirements.txt' file to specify Python dependencies, including the note-python library
- Common hardware configurations include Raspberry Pi, PC/Mac development machines, or Python-capable embedded systems. Where sensors are concerned, always default to using the I2C interface, if possible.

## Requirements

- Always use templates for notes when sending data to minimize bandwidth usage.
- Use the note-python library for all Notecard interactions.

## Suggestions

- Do not introduce power management features until the user has confirmed that the application is working. Offer this as a follow up change.
- Start with console debugging output to demonstrate that the application is working. After the user has confirmed that the application is working, logging can be adjusted.
- If the user asks for their data to be uploaded at a specific interval, ensure to set the `mode` to `periodic` in the `hub.set` request and the `outbound` to their desired interval.
- Use proper error handling and exception management when communicating with the Notecard.

## Example Basic Project

Use this example to get started with building a Python project that uses the Notecard.

You will need to know the following before starting:

- REQUIRED: The Product Unique Identifier for your application. This is a unique identifier for your application that is used to identify your Notehub project in the Notecard.
- REQUIRED (if not using I2C): The Notecard's serial port. This is the serial port that the Notecard is connected to (e.g., '/dev/ttyUSB0' on Linux or 'COM3' on Windows).
- REQUIRED (if not using serial): The I2C bus. On Raspberry Pi, this is typically bus 1.

```python
#!/usr/bin/env python3
"""
This example demonstrates the basic usage of the note-python library
to communicate with the Blues Notecard, configure it, and send sensor data.
"""

import notecard
import time
import sys

# Product UID for your Notehub project
# This should be in the format "com.company.username:productname"
# You must register at notehub.io to claim a product UID
PRODUCT_UID = "com.my-company.my-name:my-project"

def main():
    """Main application loop"""
    print(f"Starting Python application for {PRODUCT_UID}...")

    # Initialize Notecard
    # For I2C connection (default on Raspberry Pi):
    card = notecard.OpenI2C(0, 0, 0, debug=True)

    # For serial connection, use this instead:
    # card = notecard.OpenSerial("/dev/ttyUSB0", debug=True)

    # Configure the Notecard to connect to Notehub
    req = {"req": "hub.set"}
    req["product"] = PRODUCT_UID
    req["mode"] = "continuous"  # Use "periodic" for battery-powered devices

    try:
        rsp = card.Transaction(req)
        print("Notecard configured successfully")
    except Exception as e:
        print(f"Error configuring Notecard: {e}")
        sys.exit(1)

    # Main loop: send data every 15 seconds
    event_counter = 0
    max_events = 25

    while event_counter < max_events:
        event_counter += 1

        # Read temperature from Notecard's built-in sensor
        try:
            temp_req = {"req": "card.temp"}
            temp_rsp = card.Transaction(temp_req)
            temperature = temp_rsp.get("value", 0)
        except Exception as e:
            print(f"Error reading temperature: {e}")
            temperature = 0

        # Read voltage from Notecard
        try:
            volt_req = {"req": "card.voltage"}
            volt_rsp = card.Transaction(volt_req)
            voltage = volt_rsp.get("value", 0)
        except Exception as e:
            print(f"Error reading voltage: {e}")
            voltage = 0

        # Send note to Notehub
        try:
            note_req = {"req": "note.add"}
            note_req["sync"] = True  # Upload immediately for demonstration
            note_req["body"] = {
                "temp": temperature,
                "voltage": voltage,
                "count": event_counter
            }
            card.Transaction(note_req)
            print(f"Sent event {event_counter}: temp={temperature:.2f}°C, voltage={voltage:.2f}V")
        except Exception as e:
            print(f"Error sending note: {e}")

        # Wait before next reading
        time.sleep(15)

    print("Demo cycle complete. Program finished.")

if __name__ == "__main__":
    main()
```

This basic example demonstrates how to use the note-python library to send and receive JSON commands and responses.
It does not make use of the Notecard's templated note feature, see below for more information.

## Installation

Install the note-python library:

```bash
pip install note-python
```

Or add it to your `requirements.txt`:

```txt
note-python>=1.2.0
```

## Use Templates

When sending data repeatedly with the same structure, always use note templates to reduce bandwidth usage. Templates are defined once and then referenced in subsequent notes, sending only the data values without field names.

Example of setting up a template:

```python
# Define the template structure
template_req = {"req": "note.template"}
template_req["file"] = "sensors.qo"
template_req["body"] = {
    "temp": 14.1,
    "humidity": 12.1,
    "pressure": 14.1
}

try:
    card.Transaction(template_req)
    print("Template created successfully")
except Exception as e:
    print(f"Error creating template: {e}")
```

After defining the template, send notes with just the values:

```python
# Send note using the template
note_req = {"req": "note.add"}
note_req["file"] = "sensors.qo"
note_req["body"] = {
    "temp": 23.5,
    "humidity": 65.2,
    "pressure": 1013.25
}

card.Transaction(note_req)
```

## Connection Options

The note-python library supports both I2C and serial connections:

### I2C Connection (Recommended for Raspberry Pi)

```python
card = notecard.OpenI2C(0, 0, 0, debug=True)
```

### Serial Connection

```python
# Linux/Mac
card = notecard.OpenSerial("/dev/ttyUSB0", debug=True)

# Windows
card = notecard.OpenSerial("COM3", debug=True)
```
