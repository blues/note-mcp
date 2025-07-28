package main

import "github.com/mark3labs/mcp-go/mcp"

func CreateArduinoNotePowerManagementTool() mcp.Tool {
	return mcp.NewTool("arduino_note_power_management",
		mcp.WithDescription("Explain how to manage power for an Arduino project that uses the Notecard. This REQUIRES a Notecarrier-F or equivalent-wired carrier board, as the host MCU's power rails are controlled by the carrier board. If you cannot confirm this, please ask the user to confirm that they are using a Notecarrier-F or equivalent-wired carrier board."),
	)
}

func CreateArduinoNoteBestPracticesTool() mcp.Tool {
	return mcp.NewTool("arduino_note_best_practices",
		mcp.WithDescription("Explain how to write an Arduino project that uses the Notecard, using the best practices for the Notecard."),
	)
}

func CreateArduinoNoteTemplatesTool() mcp.Tool {
	return mcp.NewTool("arduino_note_templates",
		mcp.WithDescription("Explain how to format a templated note for an Arduino project."),
	)
}

func CreateArduinoCLICompileTool() mcp.Tool {
	return mcp.NewTool("arduino_compile",
		mcp.WithDescription("Compile an Arduino project targetting a specific board. This does not upload the sketch to the board."),
		mcp.WithString("board",
			mcp.Required(),
			mcp.Description("The Blues Feather MCU to build for. Valid options are: Swan, Cygnet"),
			mcp.DefaultString("Swan"),
		),
		mcp.WithString("ino",
			mcp.Required(),
			mcp.Description("The Arduino sketch to build. This is the main sketch file for the project. Use an absolute path to the sketch file. It will also need to be in a directory of the same name as the project, e.g. 'app/app.ino'"),
		),
		mcp.WithString("output-dir",
			mcp.Required(),
			mcp.Description("The directory to output the compiled sketch to. This is the directory that the compiled sketch will be saved to. Use an absolute path to a build directory in the current workspace ($PWD/build). Ensure that you have write permissions to this directory."),
		),
	)
}

func CreateArduinoCLIUploadTool() mcp.Tool {
	return mcp.NewTool("arduino_upload",
		mcp.WithDescription("Upload an Arduino sketch to a specific board. Ensure arduino_compile has been run first."),
		mcp.WithString("board",
			mcp.Required(),
			mcp.Description("The Blues Feather MCU to upload to. Valid options are: Swan, Cygnet"),
			mcp.DefaultString("Swan"),
		),
		mcp.WithString("ino",
			mcp.Required(),
			mcp.Description("The Arduino sketch to upload. This is the main sketch file for the project. Use an absolute path to the sketch file. It will also need to be in a directory of the same name as the project, e.g. 'app/app.ino'"),
		),
		mcp.WithString("port",
			mcp.Description("The port to upload the sketch to. This is the port that the sketch will be uploaded to. Use an absolute path to the port, e.g. '/dev/tty.usbmodem14101'. If not specified, the tool will attempt to find the port automatically."),
		),
	)
}

func CreateArduinoSensorsTool() mcp.Tool {
	return mcp.NewTool("arduino_sensors",
		mcp.WithDescription("Suggest sensors to use in an Arduino project."),
	)
}
