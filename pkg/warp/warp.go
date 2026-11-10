package warp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/wireshell/wireshell/pkg/config"
	"github.com/wireshell/wireshell/pkg/crypto"
)

const (
	CloudflareEndpoint = "162.159.192.1:2408"
	RegAPI             = "https://api.cloudflareclient.com/v0a2405/reg"
	DefaultServerKey   = "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo="
)

// IPInfo holds details about the current external IP
type IPInfo struct {
	Query       string `json:"query"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	RegionName  string `json:"regionName"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	Org         string `json:"org"`
	Status      string `json:"status"`
}

type warpRegRequest struct {
	Key          string `json:"key"`
	InstallID    string `json:"install_id"`
	FCMToken     string `json:"fcm_token"`
	TOS          string `json:"tos"`
	Model        string `json:"model"`
	SerialNumber string `json:"serial_number"`
	Locale       string `json:"locale"`
}

type warpRegResponse struct {
	ID     string `json:"id"`
	Token  string `json:"token"`
	Config struct {
		Peers []struct {
			PublicKey string `json:"public_key"`
			Endpoint  struct {
				V4   string `json:"v4"`
				Host string `json:"host"`
			} `json:"endpoint"`
		} `json:"peers"`
		Interface struct {
			Addresses struct {
				V4 string `json:"v4"`
				V6 string `json:"v6"`
			} `json:"addresses"`
		} `json:"interface"`
	} `json:"config"`
}

// GenerateWarpConfig registers a new anonymous WireGuard account on Cloudflare's global edge network
func GenerateWarpConfig(ctx context.Context) (*config.Config, error) {
	priv, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	pub := priv.PublicKey()

	reqBody := warpRegRequest{
		Key:          pub.Base64(),
		TOS:          time.Now().UTC().Format(time.RFC3339Nano),
		Model:        "PC",
		Locale:       "en_US",
		InstallID:    "",
		FCMToken:     "",
		SerialNumber: "",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal warp req: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", RegAPI, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("new http request: %w", err)
	}

	req.Header.Set("User-Agent", "okhttp/3.12.1")
	req.Header.Set("CF-Client-Version", "a-6.3-2405")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("warp api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("warp api failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var regResp warpRegResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, fmt.Errorf("decode warp response: %w", err)
	}

	clientIPv4 := regResp.Config.Interface.Addresses.V4
	if clientIPv4 == "" {
		clientIPv4 = "172.16.0.2"
	}
	meshIP := clientIPv4 + "/32"

	serverPubKey := DefaultServerKey
	if len(regResp.Config.Peers) > 0 && regResp.Config.Peers[0].PublicKey != "" {
		serverPubKey = regResp.Config.Peers[0].PublicKey
	}

	// Resolve endpoint
	endpoint := CloudflareEndpoint

	cfg := &config.Config{
		PrivateKey: priv.Base64(),
		PublicKey:  pub.Base64(),
		MeshIP:     meshIP,
		ListenPort: 51820,
		MTU:        1280, // Cloudflare WireGuard standard MTU
		Peers: []config.PeerConfig{
			{
				PublicKey:           serverPubKey,
				MeshIP:              "172.16.0.1/32",
				AllowedIPs:          []string{"0.0.0.0/0"}, // Routes all internet traffic!
				Endpoint:            endpoint,
				PersistentKeepalive: 25,
			},
		},
	}

	return cfg, nil
}

// FetchCurrentIP checks the public IP address, ISP, and geographic location
func FetchCurrentIP(ctx context.Context) (*IPInfo, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Try ip-api.com
	req, err := http.NewRequestWithContext(ctx, "GET", "http://ip-api.com/json", nil)
	if err == nil {
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var info IPInfo
			if err := json.NewDecoder(resp.Body).Decode(&info); err == nil && info.Query != "" {
				return &info, nil
			}
		}
	}

	// Fallback to ipinfo.io
	req2, err := http.NewRequestWithContext(ctx, "GET", "https://ipinfo.io/json", nil)
	if err != nil {
		return nil, err
	}

	resp2, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	var fallback struct {
		IP      string `json:"ip"`
		City    string `json:"city"`
		Region  string `json:"region"`
		Country string `json:"country"`
		Org     string `json:"org"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&fallback); err != nil {
		return nil, err
	}

	return &IPInfo{
		Query:       fallback.IP,
		City:        fallback.City,
		RegionName:  fallback.Region,
		Country:     fallback.Country,
		ISP:         fallback.Org,
		Org:         fallback.Org,
		Status:      "success",
	}, nil
}

// ResolveHostPort resolves a host:port to IP:port
func ResolveHostPort(hostPort string) (string, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return "", err
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return "", fmt.Errorf("failed to resolve %s: %w", host, err)
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			return net.JoinHostPort(ip.String(), port), nil
		}
	}
	return net.JoinHostPort(ips[0].String(), port), nil
}
