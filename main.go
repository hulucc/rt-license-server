package main

import (
	"crypto/tls"
	_ "embed"
	"log"
	"net/http"
	"sync"
	"io"
	"encoding/json"
	"encoding/base64"
	"time"
	"regexp"
	"strings"
	"fmt"

	"github.com/NYTimes/gziphandler"
)

//go:embed server.key
var key []byte

//go:embed server.crt
var crt []byte

func newLicense(nonce string) []byte {
	license := map[string]interface{}{
		"success": true,
		"expires_at": time.Now().AddDate(30, 0, 0).UTC().Format("2006-01-02T15:04:05Z"),
		"billing_type": "stripe",
		"max_seats": 99999,
		"license_checked_at": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"nonce": nonce,
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
		},
		"available_features": []string{},
		"trial_expires_at": nil,
		"ssop_user_email": "junruzhu@lilith.com",
	}
	bs, err := json.Marshal(license)
	if err != nil {
		log.Fatal("marshal license err: ", err)
	}
	
	body := map[string]interface{}{
		"payload": shape(base64.StdEncoding.EncodeToString(bs), 60),
		"signature": "spP3mDHpDhF3NKwtig9++JJShv3HtGmUOHPHiT950F5r/NxLIM+XIfoxbnYL\nmaGdqm9aZH/qe95yBx/s0FYoeNb1w3jLd7qJm+/LfxayRQoLhrVJHQpH9XYw\nrIrTbXtS3iH7NIZiPzVlG/X1DXF6fJtsuMTjmlVoFLv3OtlYE2Fha7sZyLSa\n44CDbxOH4vigWspH6TdwUWLqMfNvkIg38L/9+Se+gAjuWEXTtVBCten1Bnv1\niEjvkrxfsL9tljfHL/+WK7BV5Ma8paXAOuXrP0l2BYmsvZrV+FOo/91ot2eE\n/+1SoWGO+hqfL9YHB/VXR3sakV+ABU+21Mv6Pcq9SQ==\n",
	}
	result, err := json.Marshal(body)
	if err != nil {
		log.Fatal("marshal license body err:", err)
	}
	return result
}

func shape(src string, width int) string {
	re := regexp.MustCompile(fmt.Sprintf(`(\S{%d})`, width)) 
	lines := re.FindAllString(src, -1)
	return strings.Join(lines, "\n")+"\n"
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
						w.Header().Set("Content-Type", "application/json")
						w.Write(body)
						return
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
	http.Handle("/v1/licensing/verify_key", gziphandler.GzipHandler(http.HandlerFunc(LicenseServer)))
	http.HandleFunc("/v2/p", EchoServer)
	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		log.Fatal("load x509keypair err: ", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		server := http.Server{Addr: ":8099"}
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
