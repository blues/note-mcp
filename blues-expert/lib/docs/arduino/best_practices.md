# Arduino Note Best Practices

When creating a new Arduino project, there are a few best practices to follow to ensure that the project is easy to maintain and extend.

## Project Structure

- The project or sketch should be in a directory of the same name as the project, e.g. 'app/app.ino' (this is required for the Arduino CLI to work correctly)
- Create a 'README.md' in the same directory as the sketch, e.g. 'app/README.md'. This should contain a description of the project, along with instruction for how to connect any sensors to the Notecard.
- Create a 'WORKFLOW.mmd' in the same directory as the sketch, e.g. 'app/WORKFLOW.mmd'. This should contain a diagram of the flow of code in the project.
- Always assume the user is using the Blues Feather MCU (e.g. Swan) and Notecarrier-F. Where sensors are concerned, always default to using the I2C interface, if possible.

## Requirements

- Always use templates for notes.

## Suggestions

- Do not introduce power management features until the user has confirmed that the sketch is working. Offer this as a follow up change.
- Start with USB Serial debugging to demostrate that the sketch is working. After the user has confirmed that the sketch is working, this can be switched off.
- If the user asks for their data to be uploaded at a specific interval, ensure to set the `mode` to `periodic` in the `hub.set` request and the `outbound` to their desired interval.

## Example Basic Project

Use this example to get started with building an Arduino project that uses the Notecard.

You will need to know the following before starting:

- REQUIRED: The Product Unique Identifier for your application. This is a unique identifier for your application that is used to identify your Notehub project in the Notecard.
- REQUIRED (if not using I2C): The Notecard's serial port. This is the serial port that the Notecard is connected to.
- OPTIONAL: The Notecard's debug port. This is the serial port that the Arduino Debug Monitor is connected to.
- REQUIRED (if not using serial): The Notecard's I2C pins. This is the I2C pins that the Notecard is connected to.

```cpp
// This example does the same function as the "basic" example, but demonstrates
// how easy it is to use the Notecard libraries to construct JSON commands and
// also to extract responses.
//
// Using the Notecard library, you can also easily set up your Arduino
// environment to "watch" JSON request and response traffic going to/from the
// Notecard on your Arduino debug port.
//
// Note that by using the Notecard library, it is also quite easy to connect the
// Notecard to a Microcontroller's I2C ports (SDA and SCL) rather than using
// Serial, in case there is no unused serial port available to use for the
// Notecard.

// Include the Arduino library for the Notecard
#include <Notecard.h>

// If the Notecard is connected to a serial port, define it here.  For example,
// if you are using the Adafruit Feather NRF52840 Express, the RX/TX pins (and
// thus the Notecard) are on Serial1. However, if you are using an M5Stack Basic
// Core IoT Development Kit, you would connect the R2 pin to the Notecard's TX
// pin, and the M5Stack's T2 pin to the Notecard's RX pin, and then would use
// Serial2.
//
// Also, you may define a debug output port where you can watch transaction as
// they are sent to and from the Notecard.  When using the Arduino IDE this is
// typically "Serial", but you can use any available port.
//
// Note that both of these definitions are optional; just prefix either line
// with `//` to remove it.
//
// - Remove `txRxPinsSerial` if you wired your Notecard using I2C SDA/SCL pins,
//   instead of serial RX/TX.
// - Remove `usbSerial` if you don't want the Notecard library to output debug
//   information.

// #define txRxPinsSerial Serial1
#define usbSerial Serial

// This is the Product Unique Identifier for your device.  This Product UID tells
// the Notecard what type of device has embedded the Notecard, and by extension
// which vendor or customer is in charge of "managing" it.  In order to set this
// value, you must first register with notehub.io and "claim" a unique product
// ID for your device. This may have already been completed by the User.
// It could be something as simple as as your email address
// in reverse, such as "com.gmail.smith.lisa:test-device" or
// "com.outlook.gates.bill.demo"

// This is the Product Unique Identifier for your device. Only define this if you have already claimed a product identifier in the Notecard.
#define myProductID PRODUCT_UID // "com.my-company.my-name:my-project"

// This is a check to ensure that there is a product identifier defined. If not, ask the user if they already have a product identifier of if they would like to create one.
#ifndef PRODUCT_UID
#define PRODUCT_UID ""
#pragma message "PRODUCT_UID is not defined in this example. Please ensure your Notecard has a product identifier set before running this example or define it in code here. More details at https://dev.blues.io/tools-and-sdks/samples/product-uid"
#endif

Notecard notecard;

// One-time Arduino initialization
void setup()
{
    // Set up for debug output (if available/defined).
#ifdef usbSerial
    // If you open Arduino's serial terminal window, you'll be able to watch
    // JSON objects being transferred to and from the Notecard for each request.
    usbSerial.begin(115200);
    const size_t usb_timeout_ms = 3000;
    for (const size_t start_ms = millis(); !usbSerial && (millis() - start_ms) < usb_timeout_ms;)
        ;

    usbSerial.println("Starting Arduino application for %s...", myProductID);

    // For low-memory platforms, don't turn on internal Notecard logs.
#ifndef NOTE_C_LOW_MEM
    notecard.setDebugOutputStream(usbSerial);
#else
#pragma message("INFO: Notecard debug logs disabled. (non-fatal)")
#endif // !NOTE_C_LOW_MEM
#endif // usbSerial

    // Initialize the physical I/O channel to the Notecard
#ifdef txRxPinsSerial
    notecard.begin(txRxPinsSerial, 9600);
#else
    notecard.begin();
#endif

    // "newRequest()" uses the bundled "J" json package to allocate a "req",
    // which is a JSON object for the request to which we will then add Request
    // arguments.  The function allocates a "req" request structure using
    // malloc() and initializes its "req" field with the type of request.
    J *req = notecard.newRequest("hub.set");

    // This command (required) causes the data to be delivered to the Project
    // on notehub.io that has claimed this Product ID (see above).
    if (myProductID[0])
    {
        JAddStringToObject(req, "product", myProductID);
    }

    // This command determines how often the Notecard connects to the service.
    // If "continuous", the Notecard immediately establishes a session with the
    // service at notehub.io, and keeps it active continuously. Due to the power
    // requirements of a continuous connection, a battery powered device would
    // instead only sample its sensors occasionally, and would only upload to
    // the service on a "periodic" basis.
    JAddStringToObject(req, "mode", "continuous");

    // Issue the request, telling the Notecard how and how often to access the
    // service.
    // This results in a JSON message to Notecard formatted like:
    //     {
    //       "req"     : "service.set",
    //       "product" : myProductID,
    //       "mode"    : "continuous"
    //     }
    // Note that `notecard.sendRequestWithRetry()` always frees the request data
    // structure, and it returns "true" if success or "false" if there is any
    // failure. It is important to use `sendRequestWithRetry()` on the first
    // message from the MCU to the Notecard, because there will always be a
    // hardware race condition on cold boot and the Notecard must be ready to
    // receive and process the message.
    notecard.sendRequestWithRetry(req, 5); // 5 seconds
}

// In the Arduino main loop which is called repeatedly, add outbound data every
// 15 seconds
void loop()
{

    // Count the simulated measurements that we send to the cloud, and stop the
    // demo before long.
    static unsigned eventCounter = 0;
    if (++eventCounter > 25)
    {
        usbSerial.println("[APP] Demo cycle complete. Program stopped. Press RESET to restart.");
        delay(10000); // 10 seconds
        return;
    }

    // Rather than simulating a temperature reading, use a Notecard request to
    // read the temp from the Notecard's built-in temperature sensor. We use
    // `requestAndResponse()` to indicate that we would like to examine the
    // response of the transaction.  This method takes a JSON data structure,
    // "request" as input, then processes it and returns a JSON data structure,
    // "response", with the response. Note that because the Notecard library
    // uses malloc(), developers must always check for `NULL` to ensure that
    // there was enough memory available on the microcontroller to satisfy the
    // allocation request.
    double temperature = 0;
    J *rsp = notecard.requestAndResponse(notecard.newRequest("card.temp"));
    if (rsp != NULL)
    {
        temperature = JGetNumber(rsp, "value");
        notecard.deleteResponse(rsp);
    }

    // Do the same to retrieve the voltage that is detected by the Notecard on
    // its `V+` pin.
    double voltage = 0;
    rsp = notecard.requestAndResponse(notecard.newRequest("card.voltage"));
    if (rsp != NULL)
    {
        voltage = JGetNumber(rsp, "value");
        notecard.deleteResponse(rsp);
    }

    // Enqueue the measurement to the Notecard for transmission to the Notehub,
    // adding the "sync" flag for demonstration purposes to upload the data
    // instantaneously. If you are looking at this on notehub.io you will see
    // the data appearing 'live'.
    J *req = notecard.newRequest("note.add");
    if (req != NULL)
    {
        JAddBoolToObject(req, "sync", true);
        J *body = JAddObjectToObject(req, "body");
        if (body != NULL)
        {
            JAddNumberToObject(body, "temp", temperature);
            JAddNumberToObject(body, "voltage", voltage);
            JAddNumberToObject(body, "count", eventCounter);
        }
        notecard.sendRequest(req);
    }

    // Delay between samples
    delay(15 * 1000); // 15 seconds
}
```

This basic example demonstrates how to use the Notecard library to send and receive JSON commands and responses.
It does not make use of the Notecard's templated note feature, see below for more information.

## Use Templates

Use the `arduino_note_templates` tool to get the best practices for formatting a notes using templates for an Arduino project.

## Using the STLinkV3 Debugger

If you want to use serial via the STLinkV3 debugger, you can use the following code:

```cpp
HardwareSerial SerialVCP(PIN_VCP_RX, PIN_VCP_TX);
#define debugSerial SerialVCP
```
