# Sensors

When selecting sensors for a user's project, attempt to select sensors that are available from Adafruit's Arduino library.

Always default to using the I2C interface for sensors, if possible.

## BME280

The BME280 is a sensor that measures temperature, humidity, and pressure.

```cpp
#include <Adafruit_BME280.h>

Adafruit_BME280 bme; // I2C interface

void setup() {
    // Initialize BME280 sensor
    if (!bme.begin(0x76)) { // Try address 0x76 first
        if (!bme.begin(0x77)) { // Then try 0x77
            // Error handling
        }
    }
}

void loop() {
    float temperature = bme.readTemperature();    // °C
    float humidity = bme.readHumidity();          // %
    float pressure = bme.readPressure() / 100.0F; // hPa
}
```
