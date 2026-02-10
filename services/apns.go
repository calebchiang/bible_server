package services

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func SendAPNSNotification(deviceToken, title, body string) error {
	teamID := os.Getenv("APNS_TEAM_ID")
	keyID := os.Getenv("APNS_KEY_ID")
	privateKeyPEM := os.Getenv("APNS_PRIVATE_KEY")
	bundleID := os.Getenv("APNS_BUNDLE_ID")
	env := os.Getenv("APNS_ENV")

	if teamID == "" || keyID == "" || privateKeyPEM == "" || bundleID == "" {
		return errors.New("missing APNs environment variables")
	}

	authToken, err := generateAPNSToken(teamID, keyID, privateKeyPEM)
	if err != nil {
		return err
	}

	host := "https://api.push.apple.com"

	switch env {
	case "sandbox", "development", "debug":
		host = "https://api.sandbox.push.apple.com"
	case "production", "prod", "":
	default:
		return fmt.Errorf("invalid APNS_ENV: %q", env)
	}

	fmt.Println("📡 APNs host:", host)

	url := fmt.Sprintf("%s/3/device/%s", host, deviceToken)

	payload := map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{
				"title": title,
				"body":  body,
			},
			"sound": "default",
		},
	}

	jsonBody, _ := json.Marshal(payload)

	fmt.Printf("📱 Device token (%d chars): %q\n", len(deviceToken), deviceToken)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("authorization", "bearer "+authToken)
	req.Header.Set("apns-topic", bundleID)
	req.Header.Set("content-type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"APNs error: status=%d body=%s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	return nil
}

func generateAPNSToken(teamID, keyID, privateKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", errors.New("failed to decode APNs private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok || ecdsaKey.Curve != elliptic.P256() {
		return "", errors.New("invalid APNs private key")
	}

	claims := jwt.MapClaims{
		"iss": teamID,
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = keyID

	return token.SignedString(ecdsaKey)
}
