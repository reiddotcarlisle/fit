package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"fit/internal/chal"

	webauthntypes "github.com/go-ctap/ctaphid/pkg/webauthntypes"
	"github.com/go-ctap/winhello"
	"github.com/go-ctap/winhello/hiddenwindow"
)

// buildVersion injected via -ldflags.
var buildVersion = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "list":
		cmdList(args)
	case "test":
		cmdTest(args)
	case "auth":
		cmdAuth(args)
	case "add-passkey":
		cmdAddPasskey(args)
	case "delete-passkey":
		cmdDeletePasskey(args)
	case "version":
		fmt.Println(buildVersion)
	default:
		printUsage()
	}
}

func printUsage() {
	exe := os.Args[0]
	fmt.Printf("Usage: %s <command> [arguments]\n", exe)
	fmt.Println("\nCommands (Windows Hello):")
	fmt.Println("  list [--rp RP]         List Windows Hello platform credentials (filter by RP).")
	fmt.Println("  test [--rp RP]         Show diagnostic info and list credentials (subset).")
	fmt.Println("  auth --rp RP [--cred-id-hex HEX|--cred-id-b64 B64URL|--cred-index N|--device]")
	fmt.Println("                         Perform an assertion. If no allow list and --device set, Windows selector opens for external keys.")
	fmt.Println("  add-passkey --rp RP [--user USER] [--display NAME] [--device] [--resident|--no-resident]")
	fmt.Println("                         Create a new passkey. --device prefers external security keys.")
	fmt.Println("  delete-passkey [--rp RP] (--cred-id-hex HEX|--cred-id-b64 B64URL|--cred-index N)")
	fmt.Println("                         Delete a Windows Hello platform credential.")
	fmt.Println("  version                Print build version.")
	fmt.Println("\nGlobal:")
	fmt.Println("  --json                 Output JSON where applicable.")
}

// Helpers
func getString(args []string, name string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}
func getInt(args []string, name string) (int, bool) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			v, err := strconv.Atoi(args[i+1])
			if err == nil {
				return v, true
			}
			return 0, false
		}
	}
	return 0, false
}
func has(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}
func writeJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("json: %v", err)
	}
	os.Stdout.Write(b)
	os.Stdout.WriteString("\n")
}

// Windows Hello window handle
func getHelloWindow() (*hiddenwindow.HiddenWindow, func()) {
	logger := slog.New(slog.DiscardHandler)
	wnd, err := hiddenwindow.New(logger, "fit-hello")
	if err != nil {
		log.Fatalf("hidden window: %v", err)
	}
	cleanup := func() { wnd.Close() }
	return wnd, cleanup
}

func helloAttachment(args []string) winhello.WinHelloAuthenticatorAttachment {
	if has(args, "--device") {
		return winhello.WinHelloAuthenticatorAttachmentCrossPlatform
	}
	return winhello.WinHelloAuthenticatorAttachmentPlatform
}
func helloCredentialHints(args []string) []webauthntypes.PublicKeyCredentialHint {
	if has(args, "--device") {
		return []webauthntypes.PublicKeyCredentialHint{webauthntypes.PublicKeyCredentialHintSecurityKey}
	}
	return nil
}

// clientData factory
func clientData(typ, rpID string, challenge []byte) []byte {
	b64url := base64.RawURLEncoding.EncodeToString(challenge)
	origin := rpID
	if !strings.Contains(origin, "://") {
		origin = "https://" + rpID
	}
	cd := map[string]any{"type": typ, "challenge": b64url, "origin": origin, "crossOrigin": false}
	bs, err := json.Marshal(cd)
	if err != nil {
		log.Fatalf("marshal clientData: %v", err)
	}
	return bs
}

// Commands
func cmdList(args []string) {
	rp := getString(args, "--rp")
	creds, err := winhello.PlatformCredentialList(rp, false)
	if err != nil {
		log.Fatalf("PlatformCredentialList: %v", err)
	}
	if has(args, "--json") {
		items := make([]map[string]any, 0, len(creds))
		for i, c := range creds {
			items = append(items, map[string]any{
				"index":     i,
				"rp":        c.RP.ID,
				"user":      c.User.Name,
				"removable": c.Removable,
				"backedUp":  c.BackedUp,
				"credID":    base64.RawURLEncoding.EncodeToString(c.CredentialID),
			})
		}
		writeJSON(map[string]any{"backend": "hello", "rp": rp, "credentials": items})
		return
	}
	if len(creds) == 0 {
		if rp != "" {
			fmt.Printf("No credentials for RP '%s'\n", rp)
		} else {
			fmt.Println("No credentials found.")
		}
		return
	}
	fmt.Println("Windows Hello credentials:")
	for i, c := range creds {
		fmt.Printf("  [%d] RP=%s User=%s Removable=%v BackedUp=%v CredID(b64url)=%s\n", i, c.RP.ID, c.User.Name, c.Removable, c.BackedUp, base64.RawURLEncoding.EncodeToString(c.CredentialID))
	}
}

func cmdTest(args []string) {
	rp := getString(args, "--rp")
	creds, err := winhello.PlatformCredentialList(rp, false)
	if err != nil {
		log.Fatalf("PlatformCredentialList: %v", err)
	}
	if has(args, "--json") {
		items := make([]map[string]any, 0, len(creds))
		for i, c := range creds {
			if i >= 50 {
				break
			}
			items = append(items, map[string]any{
				"rp":        c.RP.ID,
				"user":      c.User.Name,
				"removable": c.Removable,
				"backedUp":  c.BackedUp,
				"credID":    base64.RawURLEncoding.EncodeToString(c.CredentialID),
			})
		}
		writeJSON(map[string]any{"backend": "hello", "rp": rp, "credentials": items, "apiVersion": winhello.APIVersionNumber()})
		return
	}
	fmt.Println("Windows Hello diagnostic:")
	fmt.Printf("  API Version: %d\n", winhello.APIVersionNumber())
	if uvpa, err := winhello.IsUserVerifyingPlatformAuthenticatorAvailable(); err == nil {
		fmt.Printf("  UVPA Available: %v\n", uvpa)
	}
	if rp != "" {
		fmt.Printf("  RP filter: %s\n", rp)
	}
	fmt.Printf("  Credentials found: %d\n", len(creds))
	max := len(creds)
	if max > 5 {
		max = 5
	}
	for i := 0; i < max; i++ {
		c := creds[i]
		fmt.Printf("    [%d] RP=%s User=%s Removable=%v BackedUp=%v CredID(b64url)=%s\n", i, c.RP.ID, c.User.Name, c.Removable, c.BackedUp, base64.RawURLEncoding.EncodeToString(c.CredentialID))
	}
	if len(creds) > max {
		fmt.Printf("    ... and %d more\n", len(creds)-max)
	}
}

func cmdAuth(args []string) {
	rp := getString(args, "--rp")
	if rp == "" {
		fmt.Println("Usage: auth --rp RP [--cred-id-hex HEX|--cred-id-b64 B64URL|--cred-index N|--device]")
		return
	}

	wnd, closeWnd := getHelloWindow()
	defer closeWnd()

	// determine credential
	var credID []byte
	if hexStr := getString(args, "--cred-id-hex"); hexStr != "" {
		b, err := hex.DecodeString(strings.TrimSpace(hexStr))
		if err != nil {
			log.Fatalf("--cred-id-hex: %v", err)
		}
		credID = b
	} else if b64 := getString(args, "--cred-id-b64"); b64 != "" {
		b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(b64))
		if err != nil {
			log.Fatalf("--cred-id-b64: %v", err)
		}
		credID = b
	} else if idx, ok := getInt(args, "--cred-index"); ok {
		list, err := winhello.PlatformCredentialList(rp, false)
		if err != nil {
			log.Fatalf("PlatformCredentialList: %v", err)
		}
		if idx < 0 || idx >= len(list) {
			log.Fatalf("--cred-index out of range (have %d)", len(list))
		}
		credID = list[idx].CredentialID
	} else if has(args, "--device") {
		// leave credID nil to open OS selector for external keys
	} else {
		list, err := winhello.PlatformCredentialList(rp, false)
		if err != nil {
			log.Fatalf("PlatformCredentialList: %v", err)
		}
		if len(list) == 0 {
			fmt.Println("No Windows Hello credentials found for RP.")
			return
		}
		credID = list[0].CredentialID
		fmt.Println("Using platform credential index 0")
	}

	challenge := chal.Bytes(32)
	cd := clientData("webauthn.get", rp, challenge)
	var allow []webauthntypes.PublicKeyCredentialDescriptor
	if len(credID) > 0 {
		allow = []webauthntypes.PublicKeyCredentialDescriptor{{Type: webauthntypes.PublicKeyCredentialTypePublicKey, ID: credID}}
	}
	asrt, err := winhello.GetAssertion(wnd.WindowHandle(), rp, cd, allow, nil, &winhello.AuthenticatorGetAssertionOptions{
		UserVerificationRequirement: winhello.WinHelloUserVerificationRequirementDiscouraged,
		AuthenticatorAttachment:     helloAttachment(args),
		CredentialHints:             helloCredentialHints(args),
		Timeout:                     30 * time.Second,
	})
	if err != nil {
		log.Fatalf("GetAssertion: %v", err)
	}

	if has(args, "--json") {
		out := map[string]any{"backend": "hello", "rp": rp, "credentialID": base64.RawURLEncoding.EncodeToString(asrt.Credential.ID), "signature": base64.RawURLEncoding.EncodeToString(asrt.Signature), "challengeB64": base64.RawURLEncoding.EncodeToString(challenge), "challengeHex": hex.EncodeToString(challenge)}
		if asrt.ExtensionOutputs != nil && asrt.ExtensionOutputs.PRFOutputs != nil && asrt.ExtensionOutputs.PRFOutputs.PRF.Enabled {
			out["prfFirst"] = base64.RawURLEncoding.EncodeToString(asrt.ExtensionOutputs.PRFOutputs.PRF.Results.First)
		}
		writeJSON(out)
	} else {
		fmt.Println("Assertion result (Hello):")
		fmt.Printf("  CredentialID(b64url): %s\n", base64.RawURLEncoding.EncodeToString(asrt.Credential.ID))
		fmt.Printf("  Sig(b64url):          %s\n", base64.RawURLEncoding.EncodeToString(asrt.Signature))
		fmt.Printf("  Challenge(b64url):    %s\n", base64.RawURLEncoding.EncodeToString(challenge))
		fmt.Printf("  Challenge(hex):       %s\n", hex.EncodeToString(challenge))
	}
}

func cmdAddPasskey(args []string) {
	rp := getString(args, "--rp")
	if rp == "" {
		fmt.Println("Usage: add-passkey --rp RP [--user USER] [--display NAME] [--device] [--resident|--no-resident]")
		return
	}
	user := getString(args, "--user")
	if user == "" {
		user = "fit-user"
	}
	display := getString(args, "--display")
	if display == "" {
		display = user
	}
	resident := true
	if has(args, "--no-resident") {
		resident = false
	}

	wnd, closeWnd := getHelloWindow()
	defer closeWnd()
	challenge := chal.Bytes(32)
	cd := clientData("webauthn.create", rp, challenge)
	userID := []byte("user-id-012345678901234567890123")

	att, err := winhello.MakeCredential(
		wnd.WindowHandle(), cd,
		webauthntypes.PublicKeyCredentialRpEntity{ID: rp, Name: rp},
		webauthntypes.PublicKeyCredentialUserEntity{ID: userID, Name: user, DisplayName: display},
		[]webauthntypes.PublicKeyCredentialParameters{{Type: webauthntypes.PublicKeyCredentialTypePublicKey, Algorithm: -7}},
		nil, nil,
		&winhello.AuthenticatorMakeCredentialOptions{
			UserVerificationRequirement: winhello.WinHelloUserVerificationRequirementDiscouraged,
			AuthenticatorAttachment:     helloAttachment(args),
			PreferResidentKey:           resident,
			CredentialHints:             helloCredentialHints(args),
			Timeout:                     45 * time.Second,
		},
	)
	if err != nil {
		log.Fatalf("MakeCredential: %v", err)
	}

	if has(args, "--json") {
		writeJSON(map[string]any{"backend": "hello", "rp": rp, "user": user, "resident": att.ResidentKey, "credentialID": base64.RawURLEncoding.EncodeToString(att.CredentialID), "challengeB64": base64.RawURLEncoding.EncodeToString(challenge), "challengeHex": hex.EncodeToString(challenge)})
	} else {
		fmt.Println("Created passkey (Hello):")
		fmt.Printf("  RP: %s\n", rp)
		fmt.Printf("  User: %s\n", user)
		fmt.Printf("  ResidentKey: %v\n", att.ResidentKey)
		fmt.Printf("  CredentialID(b64url): %s\n", base64.RawURLEncoding.EncodeToString(att.CredentialID))
		fmt.Printf("  Challenge(b64url):    %s\n", base64.RawURLEncoding.EncodeToString(challenge))
		fmt.Printf("  Challenge(hex):       %s\n", hex.EncodeToString(challenge))
	}
}

func cmdDeletePasskey(args []string) {
	rp := getString(args, "--rp")
	hexStr := getString(args, "--cred-id-hex")
	b64Str := getString(args, "--cred-id-b64")
	idx, hasIdx := getInt(args, "--cred-index")

	var credID []byte
	if hexStr != "" {
		b, err := hex.DecodeString(strings.TrimSpace(hexStr))
		if err != nil {
			log.Fatalf("--cred-id-hex: %v", err)
		}
		credID = b
	} else if b64Str != "" {
		b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(b64Str))
		if err != nil {
			log.Fatalf("--cred-id-b64: %v", err)
		}
		credID = b
	} else if hasIdx {
		list, err := winhello.PlatformCredentialList(rp, false)
		if err != nil {
			log.Fatalf("PlatformCredentialList: %v", err)
		}
		if idx < 0 || idx >= len(list) {
			log.Fatalf("--cred-index out of range (have %d)", len(list))
		}
		credID = list[idx].CredentialID
	} else {
		fmt.Println("Usage: delete-passkey [--rp RP] (--cred-id-hex HEX|--cred-id-b64 B64URL|--cred-index N)")
		return
	}

	if err := winhello.DeletePlatformCredential(credID); err != nil {
		log.Fatalf("DeletePlatformCredential: %v", err)
	}
	if has(args, "--json") {
		writeJSON(map[string]any{"backend": "hello", "deleted": true, "credID": base64.RawURLEncoding.EncodeToString(credID)})
	} else {
		fmt.Println("Credential deleted.")
	}
}
