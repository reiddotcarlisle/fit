package main

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/keys-pub/go-libfido2"
)

// buildVersion is set at build time using -ldflags "-X main.buildVersion=...".
// Defaults to "dev" when not overridden.
var buildVersion = "dev"

// main is the entry point for the CLI application.
func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "list":
		cmdList(args)
	case "auth":
		cmdAuth(args)
	case "add-passkey":
		cmdAddPasskey(args)
	case "set-pin":
		cmdSetPIN(args)
	case "reset":
		cmdReset(args)
	case "info":
		cmdInfo(args)
	case "version":
		fmt.Println(buildVersion)
	default:
		printUsage()
	}
}

// printUsage displays the available commands and their usage.
func printUsage() {
	exe := os.Args[0]
	fmt.Printf("Usage: %s <command> [arguments]\n", exe)
	fmt.Println("\nCommands:")
	fmt.Println("  list            Lists attached FIDO2 devices.")
	fmt.Println("  auth --rp RP_ID [--pin PIN] [--cred-id-hex HEX|--cred-index N] [--create] [--device N|--path PATH]")
	fmt.Println("                Performs a challenge/response (assertion). If --create is set, it will create a transient")
	fmt.Println("                credential (non-resident) first, then assert using that credential.")
	fmt.Println("  add-passkey --rp RP_ID [--user USER] [--display NAME] [--resident|--no-resident] [--pin PIN] [--device N|--path PATH]")
	fmt.Println("                Creates a new passkey (discoverable credential) on a FIDO2 security key.")
	fmt.Println("  set-pin --new NEW [--old OLD] [--device N|--path PATH]")
	fmt.Println("                Sets the device PIN (initial if --old omitted, otherwise changes PIN).")
	fmt.Println("  reset         Performs a factory reset on a FIDO2 device.")
	fmt.Println("  info [--pin PIN] [--device N|--path PATH]")
	fmt.Println("                Displays device information / non-destructive diagnostics.")
	fmt.Println("  version        Prints build version (embedded via ldflags).")
	fmt.Println("\nGlobal flags:")
	fmt.Println("  --json          Output machine-readable JSON where applicable.")
}

// cmdSetPIN sets or changes the device PIN non-interactively.
func cmdSetPIN(args []string) {
	newPIN := getStringFlag(args, "--new")
	oldPIN := getStringFlag(args, "--old")
	if newPIN == "" {
		fmt.Println("Usage: set-pin --new NEW [--old OLD] [--device N|--path PATH]")
		return
	}
	if len(newPIN) < 4 {
		fmt.Println("PIN must be at least 4 characters.")
		return
	}

	dev := getDeviceWithArgs(args)
	if dev == nil {
		return
	}

	info, err := dev.Info()
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}
	present, pinSet := clientPinStatus(info.Options)
	if !present {
		fmt.Println("Device does not implement clientPin.")
	} else if pinSet {
		fmt.Println("Device reports PIN is set (clientPin=true). If changing, provide --old.")
	} else {
		fmt.Println("Device reports PIN not set yet (clientPin=false). Set initial PIN with --new.")
	}

	action := "Setting initial PIN"
	if oldPIN != "" {
		action = "Changing PIN"
	}
	fmt.Println(action + "... You may need to touch your device.")

	if err := dev.SetPIN(newPIN, oldPIN); err != nil {
		msg := strings.ToLower(err.Error())
		if oldPIN == "" && (strings.Contains(msg, "pin required") || strings.Contains(msg, "missing parameter")) {
			fmt.Println("Device reports a PIN already exists. Provide --old to change it.")
		}
		if strings.Contains(msg, "mismatch") || strings.Contains(msg, "wrong") {
			fmt.Println("Old PIN incorrect (remaining retries may decrease).")
		}
		if strings.Contains(msg, "policy") || strings.Contains(msg, "invalid") || strings.Contains(msg, "too short") || strings.Contains(msg, "length") {
			fmt.Println("PIN rejected by policy (length/complexity). Try a longer PIN (>=4, preferably 6+ digits).")
		}
		log.Fatalf("Failed to set PIN: %v", err)
	}

	fmt.Println("PIN updated successfully.")
}

// cmdAuth performs a FIDO2 assertion (challenge/response).
func cmdAuth(args []string) {
	rpID := getStringFlag(args, "--rp")
	if rpID == "" {
		fmt.Println("Usage: auth --rp RP_ID [--pin PIN] [--cred-id-hex HEX|--cred-index N] [--create] [--device N|--path PATH]")
		return
	}
	pin := getStringFlag(args, "--pin")
	create := hasFlag(args, "--create")
	credHex := getStringFlag(args, "--cred-id-hex")
	credIndex, credIndexSet := getIntFlag(args, "--cred-index")
	// libfido2 path only

	dev := getDeviceWithArgs(args)
	if dev == nil {
		return
	}

	// Step 1: determine credential ID(s)
	var credID []byte
	if credHex != "" {
		b, err := hex.DecodeString(strings.TrimSpace(credHex))
		if err != nil {
			log.Fatalf("Invalid --cred-id-hex: %v", err)
		}
		credID = b
	} else if create {
		// Create a transient (non-resident) credential
		if pin == "" {
			fmt.Println("--create requires --pin to be provided.")
			return
		}
		cdh := libfido2.RandBytes(32)
		userID := libfido2.RandBytes(32)
		attest, err := dev.MakeCredential(
			cdh,
			libfido2.RelyingParty{ID: rpID, Name: rpID},
			libfido2.User{ID: userID, Name: "fit-user"},
			libfido2.ES256,
			pin,
			&libfido2.MakeCredentialOpts{
				// Explicitly avoid resident keys by setting RK to False
				RK: libfido2.False,
			},
		)
		if err != nil {
			log.Fatalf("MakeCredential failed: %v", err)
		}
		credID = attest.CredentialID
		fmt.Printf("Created transient credential: ID=%s Type=%s\n", hex.EncodeToString(attest.CredentialID), attest.CredentialType.String())
	} else {
		// Use an existing resident credential for this RP
		creds, err := dev.Credentials(rpID, pin)
		if err != nil {
			log.Fatalf("Credentials(%s) failed: %v", rpID, err)
		}
		if len(creds) == 0 {
			fmt.Println("No resident credentials found for RP.")
			return
		}
		pick := 0
		if credIndexSet {
			pick = credIndex
		}
		if pick < 0 || pick >= len(creds) {
			log.Fatalf("--cred-index out of range (have %d)", len(creds))
		}
		credID = creds[pick].ID
		fmt.Printf("Using resident credential index %d (len=%d)\n", pick, len(credID))
	}

	// Step 2: perform assertion using the determined credential ID
	cdh := libfido2.RandBytes(32)
	assertion, err := dev.Assertion(
		rpID,
		cdh,
		[][]byte{credID},
		pin,
		&libfido2.AssertionOpts{},
	)
	if err != nil {
		log.Fatalf("Assertion failed: %v", err)
	}

	if hasFlag(args, "--json") {
		out := map[string]any{
			"backend":      "libfido2",
			"rp":           rpID,
			"credentialID": hex.EncodeToString(assertion.CredentialID),
			"signature":    hex.EncodeToString(assertion.Sig),
			"challengeHex": hex.EncodeToString(cdh),
			"challengeB64": base64.RawURLEncoding.EncodeToString(cdh),
		}
		if len(assertion.HMACSecret) > 0 {
			out["hmacSecret"] = hex.EncodeToString(assertion.HMACSecret)
		}
		if len(assertion.AuthDataCBOR) > 0 {
			out["authDataCBOR"] = hex.EncodeToString(assertion.AuthDataCBOR)
		}
		writeJSON(out)
	} else {
		fmt.Println("Assertion result:")
		fmt.Printf("  CredentialID: %s\n", hex.EncodeToString(assertion.CredentialID))
		fmt.Printf("  Sig:          %s\n", hex.EncodeToString(assertion.Sig))
		fmt.Printf("  Challenge(hex): %s\n", hex.EncodeToString(cdh))
		fmt.Printf("  Challenge(b64): %s\n", base64.RawURLEncoding.EncodeToString(cdh))
		if len(assertion.HMACSecret) > 0 {
			fmt.Printf("  HMACSecret:   %s\n", hex.EncodeToString(assertion.HMACSecret))
		}
		if len(assertion.AuthDataCBOR) > 0 {
			fmt.Printf("  AuthDataCBOR: %s\n", hex.EncodeToString(assertion.AuthDataCBOR))
		}
	}
}

// cmdAddPasskey creates a new passkey for the given RP (resident credential by default).
func cmdAddPasskey(args []string) {
	rpID := getStringFlag(args, "--rp")
	if rpID == "" {
		fmt.Println("Usage: add-passkey --rp RP_ID [--user USER] [--display NAME] [--resident|--no-resident] [--pin PIN] [--device N|--path PATH]")
		return
	}
	userName := getStringFlag(args, "--user")
	if userName == "" {
		userName = "fit-user"
	}
	userDisplay := getStringFlag(args, "--display")
	if userDisplay == "" {
		userDisplay = userName
	}
	resident := true
	if hasFlag(args, "--no-resident") {
		resident = false
	}
	pin := getStringFlag(args, "--pin")

	// libfido2 only

	// libfido2: create resident or non-resident on a security key
	dev := getDeviceWithArgs(args)
	if dev == nil {
		return
	}
	if resident && pin == "" {
		fmt.Println("Resident passkey creation requires --pin for libfido2.")
		return
	}
	cdh := libfido2.RandBytes(32)
	userID := libfido2.RandBytes(32)
	att, err := dev.MakeCredential(
		cdh,
		libfido2.RelyingParty{ID: rpID, Name: rpID},
		libfido2.User{ID: userID, Name: userName},
		libfido2.ES256,
		pin,
		&libfido2.MakeCredentialOpts{RK: func() libfido2.OptionValue {
			if resident {
				return libfido2.True
			}
			return libfido2.False
		}()},
	)
	if err != nil {
		log.Fatalf("MakeCredential failed: %v", err)
	}
	if hasFlag(args, "--json") {
		writeJSON(map[string]any{
			"backend":      "libfido2",
			"rp":           rpID,
			"user":         userName,
			"resident":     resident,
			"credentialID": hex.EncodeToString(att.CredentialID),
			"challengeHex": hex.EncodeToString(cdh),
			"challengeB64": base64.RawURLEncoding.EncodeToString(cdh),
		})
	} else {
		fmt.Println("Created passkey:")
		fmt.Printf("  RP:            %s\n", rpID)
		fmt.Printf("  User:          %s\n", userName)
		fmt.Printf("  ResidentKey:   %v\n", resident)
		fmt.Printf("  CredentialID:  %s\n", hex.EncodeToString(att.CredentialID))
		fmt.Printf("  Challenge(hex): %s\n", hex.EncodeToString(cdh))
		fmt.Printf("  Challenge(b64): %s\n", base64.RawURLEncoding.EncodeToString(cdh))
	}
}

// cmdInfo runs a non-destructive diagnostic against the authenticator.
func cmdInfo(args []string) {
	// Extract optional --pin from args (kept simple)
	var pin string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--pin" {
			pin = args[i+1]
			break
		}
	}

	dev := getDeviceWithArgs(args)
	if dev == nil {
		return
	}

	fmt.Println("Fetching device information...")
	info, err := dev.Info()
	if err != nil {
		log.Fatalf("Info failed: %v", err)
	}

	typ, err := dev.Type()
	if err != nil {
		log.Printf("Type() error: %v", err)
	}
	isF2, err := dev.IsFIDO2()
	if err != nil {
		log.Printf("IsFIDO2() error: %v", err)
	}
	hid, err := dev.CTAPHIDInfo()
	if err != nil {
		// Not fatal; some transports may not provide HID info
		log.Printf("CTAPHIDInfo() error: %v", err)
	}

	if hasFlag(args, "--json") {
		out := map[string]any{
			"backend":    "libfido2",
			"type":       typ,
			"isFIDO2":    isF2,
			"versions":   info.Versions,
			"extensions": info.Extensions,
		}
		if hid != nil {
			out["ctapHID"] = map[string]any{"major": hid.Major, "minor": hid.Minor, "build": hid.Build, "flags": hid.Flags}
		}
		opts := map[string]string{}
		for _, o := range info.Options {
			opts[o.Name] = string(o.Value)
		}
		out["options"] = opts
		if rc, err := dev.RetryCount(); err == nil {
			out["pinRetryCount"] = rc
		}
		if pin != "" {
			if ci, err := dev.CredentialsInfo(pin); err == nil && ci != nil {
				out["residentKeys"] = map[string]int64{"existing": ci.RKExisting, "remaining": ci.RKRemaining}
			}
		}
		writeJSON(out)
	} else {
		fmt.Println("\nDevice summary:")
		fmt.Printf("  Type: %s  IsFIDO2: %v\n", typ, isF2)
		if hid != nil {
			fmt.Printf("  CTAP HID: v%d.%d build %d flags=0x%02x\n", hid.Major, hid.Minor, hid.Build, hid.Flags)
		}
		if len(info.Versions) > 0 {
			fmt.Printf("  Versions: %s\n", strings.Join(info.Versions, ", "))
		}
		if len(info.Extensions) > 0 {
			fmt.Printf("  Extensions: %s\n", strings.Join(info.Extensions, ", "))
		}
		if len(info.Options) > 0 {
			fmt.Printf("  Options:\n")
			for _, o := range info.Options {
				fmt.Printf("    - %s = %s\n", o.Name, o.Value)
			}
		}
		if rc, err := dev.RetryCount(); err == nil {
			fmt.Printf("  PIN Retry Count: %d\n", rc)
		}
		if pin != "" {
			if ci, err := dev.CredentialsInfo(pin); err == nil && ci != nil {
				fmt.Printf("  Resident Keys: existing=%d remaining=%d\n", ci.RKExisting, ci.RKRemaining)
			} else if err != nil {
				log.Printf("CredentialsInfo() error: %v", err)
			}
		}
		fmt.Println("\nTest completed.")
	}
}

// cmdList lists available FIDO devices.
func cmdList(args []string) {
	locs, err := libfido2.DeviceLocations()
	if err != nil {
		log.Fatalf("Failed to get device locations: %v", err)
	}
	if hasFlag(args, "--json") {
		items := make([]map[string]any, 0, len(locs))
		for i, loc := range locs {
			manu := strings.TrimSpace(loc.Manufacturer)
			prod := strings.TrimSpace(loc.Product)
			label := strings.TrimSpace(strings.Join([]string{manu, prod}, " "))
			if label == "" {
				label = "Unknown device"
			}
			items = append(items, map[string]any{
				"index": i,
				"label": label,
				"vid":   uint16(loc.VendorID),
				"pid":   uint16(loc.ProductID),
				"path":  loc.Path,
			})
		}
		writeJSON(map[string]any{"backend": "libfido2", "devices": items})
	} else {
		if len(locs) == 0 {
			fmt.Println("No FIDO2 devices found.")
			return
		}
		fmt.Println("Detected FIDO devices:")
		for i, loc := range locs {
			manu := strings.TrimSpace(loc.Manufacturer)
			prod := strings.TrimSpace(loc.Product)
			label := strings.TrimSpace(strings.Join([]string{manu, prod}, " "))
			if label == "" {
				label = "Unknown device"
			}
			fmt.Printf("  [%d] %s  VID:PID=%04x:%04x  Path=%s\n", i, label, uint16(loc.VendorID), uint16(loc.ProductID), loc.Path)
		}
	}
}

// cmdInit initializes the FIDO2 library.
// cmdReset performs a factory reset on a FIDO2 device.
func cmdReset(args []string) {
	fmt.Println("WARNING: This will perform a factory reset on a FIDO2 device.")
	fmt.Println("This is a destructive and irreversible action that will wipe all credentials.")
	fmt.Print("Are you sure you want to proceed? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	confirmation, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(confirmation)) != "yes" {
		fmt.Println("Aborting reset.")
		return
	}

	dev := getDeviceWithArgs(args)
	if dev == nil {
		return
	}

	fmt.Println("Performing device reset. You may need to touch your device now.")
	if err := dev.Reset(); err != nil {
		log.Fatalf("Failed to reset device: %v", err)
	}

	fmt.Println("Device has been successfully reset.")
}

// cmdChangePIN changes the PIN for a FIDO2 device.
// removed interactive init and change-pin; use set-pin instead

// getDeviceWithArgs tries to select a device using optional args:
//
//	--device N  : select by index
//	--path PATH : select by device path
//
// If not provided, auto-selects when only one device exists, otherwise prompts.
func getDeviceWithArgs(args []string) *libfido2.Device {
	locs, err := libfido2.DeviceLocations()
	if err != nil {
		log.Fatalf("Failed to get device locations: %v", err)
	}
	if len(locs) == 0 {
		log.Fatalf("No FIDO2 devices found.")
	}

	// Attempt non-interactive selectors first.
	idx, path, ok := parseDeviceSelectors(args)
	if ok {
		if path != "" {
			dev, err := libfido2.NewDevice(path)
			if err != nil {
				log.Fatalf("Failed to open device: %v", err)
			}
			return dev
		}
		if idx != nil {
			if *idx < 0 || *idx >= len(locs) {
				log.Fatalf("Invalid device index: %d", *idx)
			}
			dev, err := libfido2.NewDevice(locs[*idx].Path)
			if err != nil {
				log.Fatalf("Failed to open device: %v", err)
			}
			return dev
		}
	}

	// Auto-select if there is exactly one device.
	if len(locs) == 1 {
		dev, err := libfido2.NewDevice(locs[0].Path)
		if err != nil {
			log.Fatalf("Failed to open device: %v", err)
		}
		return dev
	}

	// Fallback to interactive selection.
	fmt.Println("Found FIDO2 devices:")
	for i, loc := range locs {
		label := strings.TrimSpace(strings.Join([]string{loc.Manufacturer, loc.Product}, " "))
		if label == "" {
			label = "Unknown device"
		}
		fmt.Printf("  [%d] %s (Path: %s)\n", i, label, loc.Path)
	}
	fmt.Print("Select a device (enter number): ")
	var index int
	_, err = fmt.Scanln(&index)
	if err != nil || index < 0 || index >= len(locs) {
		log.Fatalf("Invalid selection: %v", err)
	}
	dev, err := libfido2.NewDevice(locs[index].Path)
	if err != nil {
		log.Fatalf("Failed to open device: %v", err)
	}
	return dev
}

// parseDeviceSelectors extracts --device N or --path PATH from args.
func parseDeviceSelectors(args []string) (idx *int, path string, ok bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--device":
			if i+1 < len(args) {
				var v int
				if _, err := fmt.Sscanf(args[i+1], "%d", &v); err == nil {
					idx = &v
					ok = true
				}
				i++
			}
		case "--path":
			if i+1 < len(args) {
				path = args[i+1]
				ok = true
				i++
			}
		default:
			// ignore
		}
	}
	return
}

// getStringFlag returns the value following a named flag (e.g., --rp value).
func getStringFlag(args []string, name string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

// hasFlag returns true if the flag name is present in args.
func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

// getIntFlag parses an int following a named flag.
func getIntFlag(args []string, name string) (int, bool) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			if v, err := strconv.Atoi(args[i+1]); err == nil {
				return v, true
			}
			return 0, false
		}
	}
	return 0, false
}

// clientPinStatus returns (present, set) for the clientPin option.
// Presence means the authenticator uses a PIN; value=true means PIN set, false means not set.
func clientPinStatus(opts []libfido2.Option) (bool, bool) {
	for _, o := range opts {
		if strings.EqualFold(o.Name, "clientPin") {
			return true, o.Value == libfido2.True
		}
	}
	return false, false
}

// getHelloWindow creates a hidden window required by Windows Hello APIs.
// (Hello helpers removed in libfido2-only CLI)

// writeJSON pretty-prints JSON to stdout.
func writeJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("json: %v", err)
	}
	os.Stdout.Write(b)
	os.Stdout.WriteString("\n")
}
