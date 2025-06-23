# Arduino Notecard Power Management

IMPORTANT: This ONLY applies to the Notecard, when used in conjunction with a Notecarrier-F (or equivalently-wired carrier board that controls the host MCU's power rails).

## Example Code

```cpp
// This tutorial requires a Notecarrier-F (or equivalently-wired carrier board)
// designed enable the Notecard's ATTN pin to control a host MCU's power supply.

#include <Notecard.h>

// This is the unique Product Identifier for your device
#ifndef PRODUCT_UID
#define PRODUCT_UID "" // "com.my-company.my-name:my-project"
#pragma message "PRODUCT_UID is not defined in this example. Please ensure your Notecard has a product identifier set before running this example or define it in code here. More details at https://dev.blues.io/tools-and-sdks/samples/product-uid"
#endif

// Parameters for this example
#define myProductID PRODUCT_UID
#define notehubUploadPeriodMins 10
#define hostSleepSeconds 60

// Note that both of these definitions are optional; just prefix either line
// with `//` to remove it.
//
// - Remove `txRxPinsSerial` if you wired your Notecard using I2C SDA/SCL pins,
//   instead of serial RX/TX (I2C is the default on most Notecarrier boards).
// - Remove `usbSerial` if you don't want the Notecard library to output debug
//   information.

// #define txRxPinsSerial Serial1
#define usbSerial Serial

// Notecard I2C port definitions
Notecard notecard;

// When the Notecard puts the host MCU to sleep, it enables the host to save
// 'state' inside the Notecard while it's asleep, and to retrieve this state
// when it awakens.  These are several 'segments' of state that may individually
// be saved.
struct
{
    int cycles;
} globalState;
const char globalSegmentID[] = "GLOB";

struct
{
    int measurements;
} tempSensorState;
const char tempSensorSegmentID[] = "TEMP";

struct
{
    int measurements;
} voltSensorState;
const char voltSensorSegmentID[] = "VOLT";

// One-time Arduino initialization
void setup()
{
    // Set up for debug output (if available).
#ifdef usbSerial
    // If you open Arduino's serial terminal window, you'll be able to watch
    // JSON objects being transferred to and from the Notecard for each request.
    usbSerial.begin(115200);
    const size_t usb_timeout_ms = 3000;
    for (const size_t start_ms = millis(); !usbSerial && (millis() - start_ms) < usb_timeout_ms;)
        ;

    // For low-memory platforms, don't turn on internal Notecard logs.
#ifndef NOTE_C_LOW_MEM
    notecard.setDebugOutputStream(usbSerial);
#else
#pragma message("INFO: Notecard debug logs disabled. (non-fatal)")
#endif // !NOTE_C_LOW_MEM
#endif // usbSerial


    // Initialize the physical I2C I/O channel to the Notecard
    notecard.begin();

    // Determine whether or not this is a 'clean boot', or if we're
    // restarting after having been put to sleep by the Notecard.
    NotePayloadDesc payload;
    bool retrieved = NotePayloadRetrieveAfterSleep(&payload);

    // If the payload was successfully retrieved, attempt to restore state from
    // the payload
    if (retrieved)
    {
        // Restore the various state data structures
        retrieved &= NotePayloadGetSegment(&payload, globalSegmentID, &globalState, sizeof(globalState));
        retrieved &= NotePayloadGetSegment(&payload, tempSensorSegmentID, &tempSensorState, sizeof(tempSensorState));
        retrieved &= NotePayloadGetSegment(&payload, voltSensorSegmentID, &voltSensorState, sizeof(voltSensorState));

        // We're done with the payload, so we can free it
        NotePayloadFree(&payload);
    }

    // If this is our first time through, initialize the Notecard and state
    if (!retrieved)
    {

        // Initialize operating state
        memset(&globalState, 0, sizeof(globalState));
        memset(&tempSensorState, 0, sizeof(tempSensorState));
        memset(&voltSensorState, 0, sizeof(voltSensorState));

        // Initialize the Notecard
        J *req = notecard.newRequest("hub.set");
        if (myProductID[0])
        {
            JAddStringToObject(req, "product", myProductID);
        }
        JAddStringToObject(req, "mode", "periodic");
        JAddNumberToObject(req, "outbound", notehubUploadPeriodMins);
        notecard.sendRequestWithRetry(req, 5); // 5 seconds

        // Because many devs will be using oscilloscopes or joulescopes to
        // closely examine power consumption, it can be helpful during
        // development to provide a stable and repeatable power consumption
        // environment. In the Notecard's default configuration, the
        // accelerometer is 'on'.  As such, when debugging, devs may see tiny
        // little blips from time to time on the scope. These little blips are
        // caused by accelerometer interrupt processing, when developers
        // accidentally tap the notecard or carrier.  As such, to help during
        // development and measurement, this request disables the accelerometer.
        req = notecard.newRequest("card.motion.mode");
        JAddBoolToObject(req, "stop", true);
        notecard.sendRequest(req);
    }
}

void loop()
{
    // Bump the number of cycles
    if (++globalState.cycles > 25)
    {
        usbSerial.println("[APP] Demo cycle complete. Program stopped. Press RESET to restart.");
        delay(10000); // 10 seconds
        return;
    }

    // Simulation of a device taking a measurement of a temperature sensor.
    // Because we don't have an actual external hardware sensor in this example,
    // we're just retrieving the internal surface temperature of the Notecard.
    double currentTemperature = 0.0;
    J *rsp = notecard.requestAndResponse(notecard.newRequest("card.temp"));
    if (rsp != NULL)
    {
        currentTemperature = JGetNumber(rsp, "value");
        notecard.deleteResponse(rsp);
        tempSensorState.measurements++;
    }

    // Simulation of a device taking a measurement of a voltage sensor. Because
    // we don't have an actual external hardware sensor in this example, we're
    // just retrieving the battery voltage being supplied to the Notecard.
    double currentVoltage = 0.0;
    rsp = notecard.requestAndResponse(notecard.newRequest("card.voltage"));
    if (rsp != NULL)
    {
        currentVoltage = JGetNumber(rsp, "value");
        notecard.deleteResponse(rsp);
        voltSensorState.measurements++;
    }

    // Add a note to the Notecard containing the sensor readings
    J *req = notecard.newRequest("note.add");
    if (req != NULL)
    {
        JAddStringToObject(req, "file", "example.qo");
        J *body = JAddObjectToObject(req, "body");
        if (body != NULL)
        {
            JAddNumberToObject(body, "cycles", globalState.cycles);
            JAddNumberToObject(body, "temperature", currentTemperature);
            JAddNumberToObject(body, "temperature_measurements", tempSensorState.measurements);
            JAddNumberToObject(body, "voltage", currentVoltage);
            JAddNumberToObject(body, "voltage_measurements", voltSensorState.measurements);
        }
        notecard.sendRequest(req);
    }

    // Put ourselves back to sleep for a fixed period of time
    NotePayloadDesc payload = {0, 0, 0};
    NotePayloadAddSegment(&payload, globalSegmentID, &globalState, sizeof(globalState));
    NotePayloadAddSegment(&payload, voltSensorSegmentID, &voltSensorState, sizeof(voltSensorState));
    NotePayloadAddSegment(&payload, tempSensorSegmentID, &tempSensorState, sizeof(tempSensorState));
    NotePayloadSaveAndSleep(&payload, hostSleepSeconds, NULL);

    // We should never return here, because the Notecard put us to sleep. If we
    // do get here, it's because the Notecarrier was configured to supply power
    // to this host MCU without being switched by the ATTN pin.
    delay(15000);
}
```
