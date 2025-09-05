package main

import (
	"fmt"
	"log"
	"syscall"

	"github.com/keys-pub/go-libfido2"
	"golang.org/x/term"
)

func main() {
	// 1. Initialize the FIDO2 library. This is a mandatory first step.
	fmt.Println("Initializing FIDO2 library...")
	// No explicit initialization required for go-libfido2.

	// 2. Find and select a FIDO2 device.
	dev := getDevice()
	if dev == nil {
		return
	}
	// No need to close the device explicitly as Close() is unexported.

	// 3. Get device info to check if a PIN is already set.
	info, err := dev.Info()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	fmt.Println("Device options:")
	for _, opt := range info.Options {
		fmt.Printf("  %s: %v\n", opt.Name, opt.Value)
	}

	// 4. Check if a PIN is required but not yet set.
	if hasOption(info.Options, "clientPin") && hasOption(info.Options, "noPin") {
		fmt.Println("FIDO2 device found and requires a PIN, but none is set.")

		// 5. Prompt for a new PIN.
		fmt.Print("Enter new PIN: ")
		newPIN, err := readPassword()
		if err != nil {
			log.Fatalf("Failed to read new PIN: %v", err)
		}
		fmt.Println()

		fmt.Print("Confirm new PIN: ")
		confirmPIN, err := readPassword()
		if err != nil {
			log.Fatalf("Failed to read confirmation PIN: %v", err)
		}
		fmt.Println()

		if newPIN != confirmPIN {
			fmt.Println("PINs do not match. Aborting.")
			return
		}

		// 6. Call SetPIN with the new PIN and an empty string for the old PIN.
		fmt.Println("Setting initial PIN...")
		// For setting the initial PIN, the 'old' parameter is an empty string.
		if err := dev.SetPIN(newPIN, ""); err != nil {
			log.Fatalf("Failed to set initial PIN: %v", err)
		}
		fmt.Println("Initial PIN successfully set.")
	} else if hasOption(info.Options, "clientPin") && !hasOption(info.Options, "noPin") {
		fmt.Println("This FIDO2 device already has a PIN set. Use the 'change-pin' command to modify it.")
	} else {
		fmt.Println("This FIDO2 device does not support PINs.")
	}
}

// getDevice lists available FIDO2 devices and prompts the user to select one.
func getDevice() *libfido2.Device {
	locs, err := libfido2.DeviceLocations()
	if err != nil {
		log.Fatalf("Failed to get device locations: %v", err)
	}
	if len(locs) == 0 {
		log.Fatalf("No FIDO2 devices found.")
	}

	fmt.Println("Found FIDO2 devices:")
	for i, loc := range locs {
		fmt.Printf("  [%d] (Path: %s)\n", i, loc.Path)
	}

	fmt.Print("Select a device (enter number): ")
	var index int
	_, err = fmt.Scanln(&index)
	if err != nil || index < 0 || index >= len(locs) {
		log.Fatalf("Invalid selection: %v", err)
	}

	path := locs[index].Path
	dev, err := libfido2.NewDevice(path)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	return dev
}

// hasOption checks if the given option is present in the options slice.
func hasOption(options []libfido2.Option, opt string) bool {
	for _, o := range options {
		if o.Name == opt {
			return true
		}
	}
	return false
}

// readPassword reads a password from stdin without echoing.
func readPassword() (string, error) {
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	return string(bytePassword), nil
}
