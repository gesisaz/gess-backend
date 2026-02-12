package mpesa

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jwambugu/mpesa-golang-sdk"
)

var client *mpesa.Mpesa

func init() {
	consumerKey := os.Getenv("MPESA_CONSUMER_KEY")
	consumerSecret := os.Getenv("MPESA_CONSUMER_SECRET")
	if consumerKey == "" || consumerSecret == "" {
		log.Println("mpesa: MPESA_CONSUMER_KEY or MPESA_CONSUMER_SECRET not set; M-PESA checkout disabled")
		return
	}

	env := mpesa.EnvironmentSandbox
	if strings.ToLower(os.Getenv("MPESA_ENV")) == "production" {
		env = mpesa.EnvironmentProduction
	}

	client = mpesa.NewApp(http.DefaultClient, consumerKey, consumerSecret, env)
	envName := os.Getenv("MPESA_ENV")
	if envName == "" {
		envName = "sandbox"
	}
	log.Println("mpesa: M-PESA client initialized (env:", envName+")")
}

// Client returns the M-PESA app client, or nil if not configured.
func Client() *mpesa.Mpesa {
	return client
}

// Enabled returns true if M-PESA is configured and available.
func Enabled() bool {
	return client != nil &&
		os.Getenv("MPESA_PASSKEY") != "" &&
		os.Getenv("MPESA_SHORTCODE") != "" &&
		os.Getenv("MPESA_CALLBACK_BASE_URL") != ""
}
