package main

import (
	"crypto/tls"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed server.key
var key []byte

//go:embed server.crt
var crt []byte

func newLicense(nonce string) []byte {
	license := map[string]interface{}{
		"success":                true,
		"expires_at":             time.Now().AddDate(30, 0, 0).UTC().Format("2006-01-02T15:04:05Z"),
		"billing_type":           "stripe",
		"max_seats":              99999,
		"license_checked_at":     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"nonce":                  nonce,
		"feature_flag_overrides": map[string]interface{}{},
		"all_features": []string{
			"UnlimitedQueries",
			"PresentationMode",
			"AuditTrail",
			"PublicApps",
			"AccessControls",
			"EditorREADME",
			"UserDocs",
			"Salesforce",
			"Styling",
			// "5SeatLimit",
			"UnlimitedModules",
			"Brand",
			"Permissions",
			"Theme",
		},
		"available_features": []string{
			"UnlimitedQueries",
			"PresentationMode",
			"AuditTrail",
			"PublicApps",
			"AccessControls",
			"EditorREADME",
			"UserDocs",
			"Salesforce",
			"Styling",
			"UnlimitedModules",
		},
		"trial_expires_at":   nil,
		"ssop_user_email":    "junruzhu@lilith.com",
	}
	bs, err := json.Marshal(license)
	if err != nil {
		log.Fatal("marshal license err: ", err)
	}

	body := map[string]interface{}{
		"payload":   shape(base64.StdEncoding.EncodeToString(bs), 60),
		"signature": "spP3mDHpDhF3NKwtig9++JJShv3HtGmUOHPHiT950F5r/NxLIM+XIfoxbnYL\nmaGdqm9aZH/qe95yBx/s0FYoeNb1w3jLd7qJm+/LfxayRQoLhrVJHQpH9XYw\nrIrTbXtS3iH7NIZiPzVlG/X1DXF6fJtsuMTjmlVoFLv3OtlYE2Fha7sZyLSa\n44CDbxOH4vigWspH6TdwUWLqMfNvkIg38L/9+Se+gAjuWEXTtVBCten1Bnv1\niEjvkrxfsL9tljfHL/+WK7BV5Ma8paXAOuXrP0l2BYmsvZrV+FOo/91ot2eE\n/+1SoWGO+hqfL9YHB/VXR3sakV+ABU+21Mv6Pcq9SQ==\n",
	}
	result, err := json.Marshal(body)
	if err != nil {
		log.Fatal("marshal license body err:", err)
	}
	return result
}

func splitBy(s string, n int) []string {
	times := len(s) / n
	last := len(s) % n
	lines := make([]string, 0, times+1)
	for i := 0; i < times; i++ {
		lines = append(lines, s[i*n:(i+1)*n])
	}
	lines = append(lines, s[n*times:n*times+last])
	return lines
}

func shape(src string, width int) string {
	lines := splitBy(src, width)
	return strings.Join(lines, "\n") + "\n"
}

func LicenseServer(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		if req.Header.Get("Content-Type") == "application/json" {
			if data, err := io.ReadAll(req.Body); err == nil {
				defer req.Body.Close()
				println(string(data))
				var args map[string]interface{}
				if err := json.Unmarshal(data, &args); err == nil {
					if nonce, ok := args["nonce"].(string); ok {
						body := newLicense(nonce)
						if f, ok := w.(http.Flusher); ok {
							w.Header().Set("Content-Type", "application/json; charset=utf-8")
							w.Header().Set("X-Content-Type-Options", "nosniff")
							w.Write(body)
							f.Flush()
							return
						}
					}
				}
			}
		}
	}
	w.WriteHeader(400)
}

func EchoServer(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		if req.Header.Get("Content-Type") == "application/json" {
			w.Write([]byte("pong"))
			return
		}
	}
	w.WriteHeader(400)
}

func main() {
	http.HandleFunc("/v1/licensing/verify_key", LicenseServer)
	http.HandleFunc("/v2/p", EchoServer)
	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		log.Fatal("load x509keypair err: ", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		server := http.Server{Addr: ":80"}
		log.Println("http server starting")
		if err := server.ListenAndServe(); err != nil {
			log.Fatal("http server err: ", err)
		}
	}()
	go func() {
		defer wg.Done()
		server := http.Server{
			Addr: ":443",
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		}
		log.Println("https server starting")
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Fatal("https server err: ", err)
		}
	}()
	wg.Wait()
}
