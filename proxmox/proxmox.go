package proxmox

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/luthermonson/go-proxmox"
)

var client = proxmox.NewClient(fmt.Sprintf("%s/api2/json", os.Getenv("PROXMOX_ADDR")),
	proxmox.WithHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}),
	proxmox.WithCredentials(&proxmox.Credentials{
		Username: os.Getenv("PROXMOX_USER"),
		Password: os.Getenv("PROXMOX_PASSWORD"),
		Realm:    "pam",
	}),
)
