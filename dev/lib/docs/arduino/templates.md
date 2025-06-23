# Arduino Note Templates

When building an application that is expected to operate over a long period of time, you'll want to ensure that bandwidth is preserved and monitored, wherever possible. The Notecard provides features that allow you to optimize the size of Notes at rest and in transit, as well as a set of usage monitoring APIs.

## Working with Note Templates

By default, the Notecard allows for maximum developer flexibility in the structure and content of Notes. As such, individual Notes in a Notefile do not share structure or schema. You can add JSON structures and payloads of any type and format to a Notefile, adding and removing fields as required by your application.

In order to provide this simplicity to developers, the design of the Notefile system is primarily memory based and designed to support no more than 100 Notes per Notefile. As long as your data needs and sync periods ensure regular uploads of data to Notehub, this limit is adequate for most applications.

Some applications, however, will need to track and stage bursts of data that may eclipse the 100 Note limit in a short period of time, and before a sync can occur. For these types of use cases, the Notecard supports using a flash-based storage system based on Note templates.

Using the `note.template` request with any `.qo` or `.qos` Notefile, developers can provide the Notecard with a schema of sorts to apply to future Notes added to the Notefile. This template acts as a hint to the Notecard that allows it to internally store data as fixed-length records rather than as flexible JSON objects, which tend to be much larger.

> Note:
> Note Templates are required for both inbound and outbound Notefiles when using Notecard LoRa or NTN mode with Starnote.

## Creating a Template

To create a template, use the file argument to specify the Notefile to which the template should be applied. Then, use the body argument to specify a template body, similar to the way you'd make a note.add request. That body must contain the name of each field expected in each note.add request, and a value that serves as the hint indicating the data type to the Notecard. Each field can be a boolean, integer, float, or string. The port argument is required on Notecard LoRa and Starnote, and is a unique integer in the range 1-100.

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 50);

J *body = JCreateObject();
JAddBoolToObject(body, "new_vals", true);
JAddNumberToObject(body, "temperature", 14.1);
JAddNumberToObject(body, "humidity", 11);
JAddStringToObject(body, "pump_state", "4");
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

The Notecard responds to `note.template` with a single `bytes` field, indicating the number of bytes that will be transmitted to Notehub, per note, before compression.

```json
{
  "bytes": 40
}
```

> Warning:
> Please note that trying to "update" an existing template's body schema by using the same file argument used previously does not overwrite the old template, but rather creates a new one. This can become an issue if you create numerous Notefile templates (>25) to accommodate changes in data from individual Notes, as you may negate the advantage of templates by filling the flash storage on the Notecard and consuming additional cellular data by transferring each new template to Notehub.

In this scenario, we recommend defining a smaller number of consistent Notefile templates, binary-encoding the data and sending it in a note.add payload argument, or not using Notefile templates at all.

You can also specify a length argument that will set the maximum length of a payload (in bytes) that can be sent in Notes for the templated Notefile. If using Notecard firmware prior to v3.2.1, the length argument is required when using a payload with a templated Notefile.

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 50);
JAddNumberToObject(req, "length", 32);

J *body = JCreateObject();
JAddBoolToObject(body, "new_vals", true);
JAddNumberToObject(body, "temperature", 14.1);
JAddNumberToObject(body, "humidity", 11);
JAddStringToObject(body, "pump_state", "4");
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

Using the same body as above, and a payload length of 32 results in a template of 72 bytes.

```json
{
  "bytes": 72
}
```

## Understanding Template Data Types

The hints in each template Note body value come with a few expectations and requirements, as well as options for advanced usage.

- Boolean values must be specified in a template as true.
- String For firmware versions prior to v3.2.1 fields must be a numeric string to specify the max length. For example, "42" for a string that can be up to 42 characters in length. As of v3.2.1 variable-length strings are supported for any field and any string can be provided when configuring the template.
- Integer fields should use a specific value to indicate their type and length based on the following:
11 - for a 1 byte signed integer (e.g. -128 to 127).
12 - for a 2 byte signed integer (e.g. -32,768 to 32,767).
13 - for a 3 byte signed integer (e.g. -8,388,608 to 8,388,607).
14 - for a 4 byte signed integer (e.g. -2,147,483,648 to 2,147,483,647).
18 - for a 8 byte signed integer (e.g. -9,223,372,036,854,775,808 to 9,223,372,036,854,775,807).
21 - for a 1 byte unsigned integer (e.g. 0 to 255). Available as of v3.3.1.
22 - for a 2 byte unsigned integer (e.g. 0 to 65535). Available as of v3.3.1.
23 - for a 3 byte unsigned integer (e.g. 0 to 16777215). Available as of v3.3.1.
24 - for a 4 byte unsigned integer (e.g. 0 to 4294967295). Available as of v3.3.1.
Float fields should also use a specific value to indicate their type and length based on the following:
12.1 - for an IEEE 754 2 byte float.
14.1 - for an IEEE 754 4 byte float.
18.1 - for an IEEE 754 8 byte float.

In `note-c` and `note-arduino`, the following data types are defined:

| Data Type        | C/C++ Definition                      | Description                        |
| ---------------- | ------------------------------------- | ---------------------------------- |
| Boolean          | TBOOL                                 | Boolean true/false value           |
| String           | TSTRINGV                              | Variable length UTF-8 text         |
| String           | TSTRING(N)                            | Fixed length UTF-8 text            |
| Integer          | TINT8, TINT16, TINT24, TINT32, TINT64 | Signed integers of various sizes   |
| Unsigned Integer | TUINT8, TUINT16, TUINT24, TUINT32     | Unsigned integers of various sizes |
| Float            | TFLOAT16, TFLOAT32, TFLOAT64          | IEEE 754 floating point numbers    |

## Using Arrays in Templates

If you're working with more complex data structures, it's possible to use arrays of data types when creating a template. The same definitions are used when assigning data types to the array.

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 50);

J *body = JCreateObject();
JAddBoolToObject(body, "new_vals", true);
JAddNumberToObject(body, "temperature", 14.1);
JAddNumberToObject(body, "humidity", 11);
JAddStringToObject(body, "pump_state", "4");

J *arr = JCreateArray();
JAddItemToArray(arr, JCreateBool(true));
JAddItemToArray(arr, JCreateNumber(14.1));
JAddItemToArray(arr, JCreateNumber(11));
JAddItemToArray(arr, JCreateString("4"));
JAddItemToObject(body, "array_vals", arr);

JAddItemToObject(req, "body", body);

NoteRequest(req);
```

Starting with Notecard Firmware v9.1.1, templated Notefiles now support variable-length arrays. Define an array in your template using a single Integer or Float, and subsequent note.add requests can include any number of elements of that same type.

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 50);

J *body = JCreateObject();
JAddBoolToObject(body, "new_vals", true);
JAddNumberToObject(body, "temperature", 14.1);
JAddNumberToObject(body, "humidity", 11);
JAddStringToObject(body, "pump_state", "4");

J *arr = JCreateArray();
JAddItemToArray(arr, JCreateNumber(14.1));
JAddItemToObject(body, "array_vals", arr);

JAddItemToObject(req, "body", body);

NoteRequest(req);
```

## Use of omitempty in Templates

When using templated Notefiles it's important to know that the Notecard and Notehub enforce the usage of the omitempty instruction when serializing JSON objects. omitempty indicates that a field should be eliminated from the serialized output of a JSON object if that field has an empty value - meaning a null, false, 0, or empty string ("").

> Note: You can bypass usage of omitempty in `note.add` requests that use templated Notefiles by using the `full`:true argument.

This directly impacts templated Notefiles, especially in the body field as they appear in Notehub. For instance, while the following body would be present in Notehub when using a non-templated note.add request:

```json
"body": {
  "alert": true,
  "warning": false,
  "temp": 23.4,
  "count": 0,
  "status": "ok",
  "prevstatus": ""
}
```

...that same body will look like this in Notehub if using a templated Notefile:

```json
"body": {
  "alert": true,
  "temp": 23.4,
  "status": "ok"
}
```

## Verifying a Template

You can use the `verify:true` argument to return the current template for a Notefile.

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");
JAddBoolToObject(req, "verify", true);

NoteRequest(req);
```

If the file provided has an active template, it will be returned in a response body.

```json
{
 "body": {
  "new_vals": true,
  "temperature": 14.1,
  "humidity": 11,
  "pump_state": "4"
 },
 "template": true,
 "length": 32
}
```

## Creating Compact Templates

By default all Note templates automatically include metadata, including a timestamp for when the Note was created, various fields about a device's location, as well as a timestamp for when the device's location was determined.

By providing the note.template request a "format" of "compact", you can tell the Notecard to omit this additional metadata to save on storage and bandwidth. The use of "format": "compact" is required for Notecard LoRa and a Notecard paired with Starnote.

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 10);
JAddStringToObject(req, "format", "compact");

J *body = JCreateObject();
JAddNumberToObject(body, "temperature", 14.1);
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

When using "compact" templates, you may include the following keywords in your template to restore selected fields to their original position in the Note (e.g. best_lat) that would otherwise be omitted:

- `_lat`: The device's latitude. For a value provide either 12.1 (2-byte float), 14.1 (4-byte float), or 18.1 (8-byte float) depending on your desired precision.
- `_lon`: The device's longitude. For a value provide either 12.1 (2-byte float), 14.1 (4-byte float), or 18.1 (8-byte float) depending on your desired precision.
- `_ltime`: A timestamp for when the device's location was determined. For a value provide 14 (4-byte integer).
- `_time`: A timestamp for when the Note was created. For a value provide 14 (4-byte integer).

For example the following template includes all additional fields.

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 10);
JAddStringToObject(req, "format", "compact");

J *body = JCreateObject();
JAddNumberToObject(body, "temperature", 14.1);
JAddNumberToObject(body, "_lat", 14.1);
JAddNumberToObject(body, "_lon", 14.1);
JAddNumberToObject(body, "_ltime", 14);
JAddNumberToObject(body, "_time", 14);
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

> Warning:
When using the string data type in a compact template, each string value in a Note is limited to a maximum of 255 characters. Notecard will return the following error if a string > 255 characters is used:

```json
{
 "err": "error adding note: compact mode only supports strings up to 255 bytes 
 {template-incompatible}"
}
```

## Using Templates with a Payload

The most efficient way to send base64-encoded binary data with Notecard is to use the payload argument in a note.add request. When using templates, you can arbitrarily add a payload to any note.add request.

You can also separately define a template that only uses a payload by sending an empty body argument (i.e. with no payload argument at all) when creating the template:

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");

J *body = JCreateObject();
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

> Warning:
Please be aware when using a payload with compact templates in NTN mode (e.g. with Starnote for Skylo), the maximum packet size is 256 bytes.

## Adding Notes to a Template Notefile

After a template is created, use `note.add` requests to create Notes that conform to the template.



```c
J *req = NoteNewRequest("note.add");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 50);

J *body = JCreateObject();
JAddBoolToObject(body, "new_vals", true);
JAddNumberToObject(body, "temperature", 22.22);
JAddNumberToObject(body, "humidity", 43);
JAddStringToObject(body, "pump_state", "off");
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

When adding Notes to a Notefile with an active template, the following JSON object is returned by the Notecard:

```json
{ "template": true }
```

Notefiles with an active template validate each Note upon a `note.add` request. If any value in the Note body does not adhere to the template, or if the payload is longer than specified, an error is returned. For instance, the following Note includes a float for the humidity, which was specified in the template as an integer.

```c
J *req = NoteNewRequest("note.add");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 50);

J *body = JCreateObject();
JAddBoolToObject(body, "new_vals", true);
JAddNumberToObject(body, "temperature", 22.22);
JAddNumberToObject(body, "humidity", 43.22); // mistakenly specified here as a float instead of integer
JAddStringToObject(body, "pump_state", "off");
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

```json
{
 "err": "error adding note: integer expected because of template"
}
```

For string values, an error is not returned on a `note.add`, but the provided value is truncated to the length (if specified in the template). For instance, the following Note includes a pump_state string longer than the maximum length defined in the template. The pump_state for this Note is truncated to four characters and saved as acti.

```c
J *req = NoteNewRequest("note.add");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 50);

J *body = JCreateObject();
JAddBoolToObject(body, "new_vals", true);
JAddNumberToObject(body, "temperature", 22.22);
JAddNumberToObject(body, "humidity", 43);
JAddStringToObject(body, "pump_state", "active"); // will be saved as "acti"
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

## Modifying a Template

If the needs of your application evolve, you can modify a template with another note.template request to the same Notefile. A new template can be set at any time and is non-destructive, meaning it has no impact on existing Notes in the Notefile.

For instance, you may need to modify the template field data types and/or add/remove fields:

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");
JAddNumberToObject(req, "port", 50);

J *body = JCreateObject();
JAddBoolToObject(body, "new_vals", true);
JAddNumberToObject(body, "temperature", 14.1); // Change to a 4 byte float
JAddNumberToObject(body, "humidity", 11);
JAddStringToObject(body, "pump_state", "4");
JAddNumberToObject(body, "pressure", 12.1); // New field
JAddItemToObject(req, "body", body);

NoteRequest(req);
```

These template changes will be applied only to new Notes in the Notefile. Existing Notes remain unchanged.

## Clearing a Template

To clear a template from a Notefile, simply call `note.template` with the Notefile name and omit the body and payload arguments. After clearing the template, all Notes written to the Notefile are stored as arbitrary JSON structures. This request, if successful, will return an empty JSON body ({}).

```c
J *req = NoteNewRequest("note.template");
JAddStringToObject(req, "file", "readings.qo");

NoteRequest(req);
```
