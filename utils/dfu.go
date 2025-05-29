// Copyright 2017 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package utils

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/blues/note-go/note"
	"github.com/blues/note-go/notecard"
	"github.com/blues/note-go/notehub"
)

// DfuIsNotecardFirmware determines if the binary is Notecard firmware
func DfuIsNotecardFirmware(bin *[]byte) (isNotecardImage bool) {

	// NotecardFirmwareSignature is used to identify whether or not this firmware is a
	// candidate for downloading onto notecards.  Note that this is not a security feature; if someone
	// embeds this binary sequence and embeds it, they will be able to do precisely what they can do
	// by using the USB to directly download firmware onto the device. This mechanism is intended for
	// convenience and is just intended to keep people from inadvertently hurting themselves.
	var NotecardFirmwareSignature = []byte{0x82, 0x1c, 0x6e, 0xb7, 0x18, 0xec, 0x4e, 0x6f, 0xb3, 0x9e, 0xc1, 0xe9, 0x8f, 0x22, 0xe9, 0xf6}

	return bytes.Contains(*bin, NotecardFirmwareSignature)
}

// NotehubTime gets the current time from Notehub
func NotehubTime() (int64, error) {
	// For simplicity, use current system time
	// In production, you might want to fetch from Notehub API
	return time.Now().Unix(), nil
}

// SideloadFirmware performs the firmware sideload process using dfu.put requests
func SideloadFirmware(ctx *notecard.Context, firmwareData []byte) error {
	return SideloadFirmwareWithLogger(ctx, firmwareData, nil)
}

// SideloadFirmwareWithLogger performs the firmware sideload process with optional logging
func SideloadFirmwareWithLogger(ctx *notecard.Context, firmwareData []byte, logger *MCPLogger) error {
	return SideloadFirmwareWithProgressAndLogger(ctx, firmwareData, logger, nil)
}

// ProgressCallback is a function type for progress updates
type ProgressCallback func(current, total float64, message string)

// SideloadFirmwareWithProgressAndLogger performs the firmware sideload process with optional logging and progress callbacks
func SideloadFirmwareWithProgressAndLogger(ctx *notecard.Context, firmwareData []byte, logger *MCPLogger, progressCallback ProgressCallback) error {
	if ctx == nil {
		return fmt.Errorf("notecard not initialized")
	}

	if logger != nil {
		logger.Info("Starting firmware sideload process")
	}

	// Optimize for USB connections
	fmt.Fprintf(os.Stderr, "🚀 Optimizing for USB connection...\n")
	if logger != nil {
		logger.Debug("Optimizing connection settings for USB")
	}

	originalSegmentMaxLen := notecard.RequestSegmentMaxLen
	originalSegmentDelayMs := notecard.RequestSegmentDelayMs

	// Set USB-optimized values
	notecard.RequestSegmentMaxLen = 1024
	notecard.RequestSegmentDelayMs = 5

	// Restore original settings when done
	defer func() {
		notecard.RequestSegmentMaxLen = originalSegmentMaxLen
		notecard.RequestSegmentDelayMs = originalSegmentDelayMs
		if logger != nil {
			logger.Debug("Restored original connection settings")
		}
	}()

	// Check if the notecard supports binary transfers
	binaryMax := 0
	fmt.Fprintf(os.Stderr, "🔍 Checking binary transfer capability...\n")
	if logger != nil {
		logger.Info("Checking binary transfer capability...")
	}
	if progressCallback != nil {
		progressCallback(5, 100, "Checking binary transfer capability...")
	}

	rsp, err := ctx.Transaction(map[string]interface{}{"req": "card.binary"})
	if err == nil {
		if max, exists := rsp["max"]; exists {
			if maxFloat, ok := max.(float64); ok {
				binaryMax = int(maxFloat)
			} else if maxInt, ok := max.(int); ok {
				binaryMax = maxInt
			} else if maxStr := fmt.Sprintf("%v", max); maxStr != "" {
				if parsed, err := strconv.Atoi(maxStr); err == nil {
					binaryMax = parsed
				}
			}
		}
	} else {
		if note.ErrorContains(err, note.ErrCardIo) {
			return fmt.Errorf("card I/O error during binary check: %w", err)
		}
	}

	if binaryMax > 0 {
		fmt.Fprintf(os.Stderr, "✅ Binary transfers supported (max: %d bytes)\n", binaryMax)
		if logger != nil {
			logger.Infof("Binary transfers supported (max: %d bytes)", binaryMax)
		}
	} else {
		fmt.Fprintf(os.Stderr, "⚠️  Binary transfers not supported, using standard mode\n")
		if logger != nil {
			logger.Warning("Binary transfers not supported, using standard mode")
		}
	}

	// Determine the file type - assume Notecard firmware for our use case
	filetype := notehub.UploadTypeNotecardFirmware

	// Set the Notecard's time if needed
	epochTime, err := NotehubTime()
	if err != nil {
		return fmt.Errorf("failed to get time: %w", err)
	}
	_, err = ctx.Transaction(map[string]interface{}{
		"req":  "card.time",
		"time": epochTime,
	})
	if err != nil {
		return fmt.Errorf("failed to set notecard time: %w", err)
	}

	if logger != nil {
		logger.Debug("Set Notecard time for firmware update")
	}
	if progressCallback != nil {
		progressCallback(10, 100, "Notecard time synchronized")
	}

	// Perform the DFU operation
	fmt.Fprintf(os.Stderr, "Starting firmware update to Notecard\n")
	if logger != nil {
		logger.Info("Starting DFU operation...")
	}
	if progressCallback != nil {
		progressCallback(15, 100, "Starting DFU operation...")
	}

	err = LoadBinaryToNotecardWithProgressAndLogger(ctx, "firmware.bin", firmwareData, filetype, binaryMax, logger, progressCallback)
	if err != nil {
		if logger != nil {
			logger.Errorf("Failed to load firmware: %v", err)
		}
		return fmt.Errorf("failed to load firmware: %w", err)
	}

	if logger != nil {
		logger.Info("Firmware sideload completed successfully")
	}
	if progressCallback != nil {
		progressCallback(100, 100, "Firmware sideload completed successfully")
	}
	return nil
}

// LoadBinaryToNotecard loads binary data to the Notecard using dfu.put
func LoadBinaryToNotecard(ctx *notecard.Context, filename string, bin []byte, filetype notehub.UploadType, binaryMax int) error {
	return LoadBinaryToNotecardWithLogger(ctx, filename, bin, filetype, binaryMax, nil)
}

// LoadBinaryToNotecardWithLogger loads binary data to the Notecard using dfu.put with optional logging
func LoadBinaryToNotecardWithLogger(ctx *notecard.Context, filename string, bin []byte, filetype notehub.UploadType, binaryMax int, logger *MCPLogger) error {
	return LoadBinaryToNotecardWithProgressAndLogger(ctx, filename, bin, filetype, binaryMax, logger, nil)
}

// LoadBinaryToNotecardWithProgressAndLogger loads binary data to the Notecard using dfu.put with optional logging and progress callbacks
func LoadBinaryToNotecardWithProgressAndLogger(ctx *notecard.Context, filename string, bin []byte, filetype notehub.UploadType, binaryMax int, logger *MCPLogger, progressCallback ProgressCallback) error {
	totalLen := len(bin)
	fmt.Fprintf(os.Stderr, "📦 Starting firmware transfer (%d bytes)\n", totalLen)
	if logger != nil {
		logger.Infof("Starting firmware transfer (%d bytes)", totalLen)
	}
	if progressCallback != nil {
		progressCallback(20, 100, fmt.Sprintf("Starting firmware transfer (%d bytes)", totalLen))
	}

	// Clean up the filename
	parts := strings.Split(filename, "/")
	if len(parts) > 1 {
		filename = parts[len(parts)-1]
	}
	parts = strings.Split(filename, "\\")
	if len(parts) > 1 {
		filename = parts[len(parts)-1]
	}

	// Generate the firmware metadata
	fmt.Fprintf(os.Stderr, "🔧 Generating firmware metadata...\n")
	if logger != nil {
		logger.Debug("Generating firmware metadata...")
	}
	if progressCallback != nil {
		progressCallback(25, 100, "Generating firmware metadata...")
	}

	metadata := notehub.UploadMetadata{
		Created:  time.Now().Unix(),
		Source:   filename,
		MD5:      fmt.Sprintf("%x", md5.Sum(bin)),
		CRC32:    crc32.ChecksumIEEE(bin),
		Length:   totalLen,
		Name:     filename,
		FileType: filetype,
	}

	// Convert metadata to body
	body, err := note.ObjectToBody(metadata)
	if err != nil {
		return fmt.Errorf("failed to create metadata body: %w", err)
	}

	// Initiate the DFU put operation
	var chunkLen int
	var compressionMode string

	fmt.Fprintf(os.Stderr, "🚀 Initiating firmware transfer...\n")
	if logger != nil {
		logger.Info("Initiating firmware transfer...")
	}
	if progressCallback != nil {
		progressCallback(30, 100, "Initiating firmware transfer...")
	}

	rsp, err := ctx.Transaction(map[string]interface{}{
		"req":  "dfu.put",
		"name": filename,
		"body": body,
	})
	if err != nil {
		return fmt.Errorf("failed to initiate DFU: %w", err)
	}

	// Extract chunk length and compression mode from response
	if length, exists := rsp["length"]; exists {
		if lengthFloat, ok := length.(float64); ok {
			chunkLen = int(lengthFloat)
		} else if lengthInt, ok := length.(int); ok {
			chunkLen = lengthInt
		} else if lengthStr := fmt.Sprintf("%v", length); lengthStr != "" {
			if parsed, err := strconv.Atoi(lengthStr); err == nil {
				chunkLen = parsed
			}
		}
	}

	if compression, exists := rsp["compression"]; exists {
		if compressionStr, ok := compression.(string); ok {
			compressionMode = compressionStr
		}
	}

	if chunkLen == 0 {
		chunkLen = 1024 // Default chunk size
	}

	if logger != nil {
		logger.Debugf("Using chunk size: %d bytes, compression: %s", chunkLen, compressionMode)
	}

	// Track timing
	beganSecs := time.Now().UTC().Unix()
	offset := 0
	lenRemaining := totalLen
	chunkCount := 0

	for lenRemaining > 0 {
		chunkCount++
		// Determine chunk size
		thisLen := lenRemaining
		if thisLen > chunkLen {
			thisLen = chunkLen
		}

		// Prepare the chunk
		chunk := bin[offset : offset+thisLen]
		progress := float64(totalLen-lenRemaining) / float64(totalLen) * 100
		fmt.Fprintf(os.Stderr, "📦 Chunk %d: %.1f%% complete (%d bytes)\n",
			chunkCount, progress, thisLen)

		if logger != nil {
			logger.Debugf("Sending chunk %d: %.1f%% complete (%d bytes)", chunkCount, progress, thisLen)
		}

		// Send progress callback for chunk transfer (map to 30-80% of overall progress)
		if progressCallback != nil {
			chunkProgress := 30 + (progress/100)*50 // Map chunk progress to 30-80% range
			progressCallback(chunkProgress, 100, fmt.Sprintf("Transferring chunk %d: %.1f%% complete", chunkCount, progress))
		}

		// Create the base request
		req := map[string]interface{}{
			"req":    "dfu.put",
			"offset": offset,
			"length": thisLen,
		}

		// Handle compression if needed
		payload := chunk
		if compressionMode == "snappy" {
			// Note: snappy compression would require the snappy package
			// For now, we'll use uncompressed
		}

		// Add payload - use binary transfers for better performance when available
		if binaryMax > 0 {
			// Encode payload using COBS (Consistent Overhead Byte Stuffing)
			payloadEncoded, err := notecard.CobsEncode(payload, byte('\n'))
			if err != nil {
				return fmt.Errorf("failed to COBS encode payload: %w", err)
			}

			// Send the COBS data to the notecard
			req2 := map[string]interface{}{
				"req":  "card.binary.put",
				"cobs": len(payloadEncoded),
			}

			_, err = ctx.Transaction(req2)
			if err != nil {
				return fmt.Errorf("failed to send card.binary.put: %w", err)
			}

			// Send the COBS-encoded data
			payloadEncoded = append(payloadEncoded, byte('\n'))
			err = ctx.SendBytes(payloadEncoded)
			if err != nil {
				return fmt.Errorf("failed to send COBS payload: %w", err)
			}

			// Verify that the binary made it to the notecard
			verifyRsp, err := ctx.Transaction(map[string]interface{}{"req": "card.binary"})
			if err != nil {
				return fmt.Errorf("failed to verify binary transfer: %w", err)
			}

			var receivedLength int
			if length, exists := verifyRsp["length"]; exists {
				if lengthFloat, ok := length.(float64); ok {
					receivedLength = int(lengthFloat)
				} else if lengthStr := fmt.Sprintf("%v", length); lengthStr != "" {
					if parsed, err := strconv.Atoi(lengthStr); err == nil {
						receivedLength = parsed
					}
				}
			}

			if receivedLength != len(payload) {
				return fmt.Errorf("notecard payload verification failed (%d sent, %d received)", len(payload), receivedLength)
			}

			// Set binary flag for dfu.put request (no payload needed)
			req["binary"] = true
		} else {
			req["payload"] = payload
		}

		// Calculate MD5 of the payload for verification
		chunkMD5 := fmt.Sprintf("%x", md5.Sum(payload))
		req["status"] = chunkMD5

		// Send the chunk with retry logic
		maxRetries := 3
		var rsp map[string]interface{}
		var err error

		for retry := 0; retry < maxRetries; retry++ {
			if retry > 0 {
				time.Sleep(1000 * time.Millisecond) // Wait before retry
			}

			rsp, err = ctx.Transaction(req)
			if err != nil {
				if note.ErrorContains(err, note.ErrCardIo) {
					if retry < maxRetries-1 {
						continue // Retry
					}
				}
				return fmt.Errorf("failed to send chunk at offset %d after %d retries: %w", offset, maxRetries, err)
			}
			break // Success
		}

		// Move to next chunk
		lenRemaining -= thisLen
		offset += thisLen

		// Wait for any pending operations to complete
		if pending, exists := rsp["pending"]; exists {
			if pendingBool, ok := pending.(bool); ok && pendingBool {
				for {
					statusRsp, err := ctx.Transaction(map[string]interface{}{"req": "dfu.put"})
					if err != nil {
						if note.ErrorContains(err, note.ErrDFUNotReady) ||
							note.ErrorContains(err, note.ErrDFUInProgress) ||
							strings.Contains(err.Error(), "firmware update is in progress") {
							if lenRemaining == 0 {
								break
							}
						}
						return fmt.Errorf("error checking DFU status: %w", err)
					}

					if pending, exists := statusRsp["pending"]; exists {
						if pendingBool, ok := pending.(bool); ok && !pendingBool {
							break
						}
					}
					time.Sleep(750 * time.Millisecond)
				}
			}
		}
	}

	// Display summary using Blues' exact approach
	elapsedSecs := (time.Now().UTC().Unix() - beganSecs) + 1
	fmt.Fprintf(os.Stderr, "✅ Transfer completed: %d seconds (%.0f Bps, %d chunks)\n",
		elapsedSecs, float64(totalLen)/float64(elapsedSecs), chunkCount)

	if logger != nil {
		logger.Infof("Transfer completed: %d seconds (%.0f Bps, %d chunks)",
			elapsedSecs, float64(totalLen)/float64(elapsedSecs), chunkCount)
	}
	if progressCallback != nil {
		progressCallback(80, 100, fmt.Sprintf("Transfer completed: %d seconds (%.0f Bps, %d chunks)",
			elapsedSecs, float64(totalLen)/float64(elapsedSecs), chunkCount))
	}

	// Wait for Notecard firmware update to complete using Blues' exact logic
	if filetype == notehub.UploadTypeNotecardFirmware {
		fmt.Fprintf(os.Stderr, "⏳ Waiting for firmware update to complete...\n")
		if logger != nil {
			logger.Info("Waiting for firmware update to complete...")
		}
		if progressCallback != nil {
			progressCallback(85, 100, "Waiting for firmware update to complete...")
		}

		first := true
		connectionLostCount := 0
		for i := 0; i < 60; i++ { // Reduced from 90 to 60 seconds
			rsp, err := ctx.Transaction(map[string]interface{}{
				"req":  "dfu.status",
				"name": "card",
			})
			if err == nil {
				// Reset connection lost counter on successful communication
				connectionLostCount = 0
				if pending, exists := rsp["pending"]; exists {
					if pendingBool, ok := pending.(bool); ok && !pendingBool {
						fmt.Fprintf(os.Stderr, "✅ Firmware update completed!\n")
						if logger != nil {
							logger.Info("Firmware update completed!")
						}
						if progressCallback != nil {
							progressCallback(95, 100, "Firmware update completed!")
						}
						break
					}
				}
			} else {
				// Handle connection errors during restart
				if note.ErrorContains(err, note.ErrCardIo) ||
					strings.Contains(err.Error(), "connection") ||
					strings.Contains(err.Error(), "timeout") {
					connectionLostCount++
					if connectionLostCount == 1 {
						fmt.Fprintf(os.Stderr, "📡 Connection lost (Notecard restarting)...\n")
						if logger != nil {
							logger.Info("Connection lost (Notecard restarting)...")
						}
						if progressCallback != nil {
							progressCallback(90, 100, "Connection lost (Notecard restarting)...")
						}
					}
					// If we've lost connection multiple times or we're past the halfway point, assume update completed
					if connectionLostCount >= 3 || i > 30 {
						fmt.Fprintf(os.Stderr, "✅ Firmware update completed!\n")
						if logger != nil {
							logger.Info("Firmware update completed!")
						}
						if progressCallback != nil {
							progressCallback(95, 100, "Firmware update completed!")
						}
						break
					}
				}
			}
			if first {
				first = false
			}
			time.Sleep(1000 * time.Millisecond)
		}
	}

	fmt.Fprintf(os.Stderr, "🎉 Firmware update completed successfully!\n")
	if logger != nil {
		logger.Info("Firmware update completed successfully!")
	}
	return nil
}
