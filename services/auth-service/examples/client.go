// Package main — example typed Go client for the auth-service.
//
//   go run ./examples/client.go \
//     -base http://localhost:8081 \
//     -email alice@example.com \
//     -password 'correct horse battery staple'
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client is a tiny typed HTTP wrapper for the auth-service.
type Client struct {
	Base   string
	HTTP   *http.Client
	Tokens Tokens
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

// Health probes the public health endpoint.
func (c *Client) Health() error {
	r, err := c.HTTP.Get(c.Base + "/healthz")
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return fmt.Errorf("health: %d", r.StatusCode)
	}
	return nil
}

// Register a new user.
func (c *Client) Register(email, password, fullName string) error {
	body, _ := json.Marshal(registerReq{email, password, fullName})
	r, err := c.HTTP.Post(c.Base+"/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 {
		b, _ := io.ReadAll(r.Body)
		return fmt.Errorf("register: %d %s", r.StatusCode, string(b))
	}
	return nil
}

// Login exchanges email/password for tokens and stores them on the client.
func (c *Client) Login(email, password string) error {
	body, _ := json.Marshal(loginReq{email, password})
	r, err := c.HTTP.Post(c.Base+"/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		b, _ := io.ReadAll(r.Body)
		return fmt.Errorf("login: %d %s", r.StatusCode, string(b))
	}
	return json.NewDecoder(r.Body).Decode(&c.Tokens)
}

// Refresh rotates the access token.
func (c *Client) Refresh() error {
	body, _ := json.Marshal(refreshReq{c.Tokens.RefreshToken})
	r, err := c.HTTP.Post(c.Base+"/v1/auth/refresh", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(&c.Tokens)
}

// Me returns the current user profile (call /v1/auth/me).
func (c *Client) Me() (map[string]any, error) {
	r, err := c.authed(http.MethodGet, "/v1/auth/me", nil)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	out := map[string]any{}
	return out, json.NewDecoder(r.Body).Decode(&out)
}

// CreateAPIKey provisions a new API key.
func (c *Client) CreateAPIKey(name string, scopes []string) (map[string]any, error) {
	body, _ := json.Marshal(map[string]any{"name": name, "scopes": scopes})
	r, err := c.authed(http.MethodPost, "/v1/auth/api-keys", body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	out := map[string]any{}
	return out, json.NewDecoder(r.Body).Decode(&out)
}

func (c *Client) authed(method, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, c.Base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Tokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	return c.HTTP.Do(req)
}

func main() {
	base := flag.String("base", "http://localhost:8081", "service base URL")
	email := flag.String("email", "", "email")
	password := flag.String("password", "", "password")
	flag.Parse()

	if *email == "" || *password == "" {
		log.Fatal("email + password required")
	}
	c := &Client{Base: strings.TrimRight(*base, "/"), HTTP: &http.Client{Timeout: 5 * time.Second}}
	if err := c.Health(); err != nil {
		log.Fatalf("health: %v", err)
	}
	fmt.Println("healthy ✓")
	if err := c.Login(*email, *password); err != nil {
		_ = c.Register(*email, *password, *email)
		if err := c.Login(*email, *password); err != nil {
			log.Fatalf("login: %v", err)
		}
	}
	fmt.Println("logged in ✓  access=", c.Tokens.AccessToken[:24]+"…")
	me, _ := c.Me()
	fmt.Printf("me: id=%v email=%v\n", me["id"], me["email"])
	k, _ := c.CreateAPIKey("smoke", []string{"read"})
	fmt.Printf("api-key created: id=%v prefix=%v\n", k["id"], k["prefix"])
}
