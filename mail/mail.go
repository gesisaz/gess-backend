package mail

import (
	"fmt"
	"log"
	"os"

	"github.com/resend/resend-go/v3"
)

const (
	TokenTypeEmailVerification = "email_verification"
	TokenTypePasswordReset    = "password_reset"
)

// getClient returns a Resend client using RESEND_API_KEY from env.
func getClient() *resend.Client {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Println("mail: RESEND_API_KEY not set; emails will not be sent")
		return nil
	}
	return resend.NewClient(apiKey)
}

// getFrom returns the from address (MAIL_FROM or Resend onboarding default).
func getFrom() string {
	from := os.Getenv("MAIL_FROM")
	if from != "" {
		return from
	}
	return "onboarding@resend.dev"
}

// SendVerificationEmail sends an email verification link to the given address.
func SendVerificationEmail(to, verificationLink string) error {
	client := getClient()
	if client == nil {
		return fmt.Errorf("mail client not configured")
	}

	subject := "Verify your email address"
	html := fmt.Sprintf(`<p>Please verify your email by clicking the link below:</p>
<p><a href="%s">Verify my email</a></p>
<p>If you did not create an account, you can ignore this email.</p>
<p>This link expires in 24 hours.</p>`, verificationLink)

	params := &resend.SendEmailRequest{
		From:    getFrom(),
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}

	_, err := client.Emails.Send(params)
	return err
}

// SendPasswordResetEmail sends a password reset link to the given address.
func SendPasswordResetEmail(to, resetLink string) error {
	client := getClient()
	if client == nil {
		return fmt.Errorf("mail client not configured")
	}

	subject := "Reset your password"
	html := fmt.Sprintf(`<p>You requested a password reset. Click the link below to set a new password:</p>
<p><a href="%s">Reset my password</a></p>
<p>If you did not request this, you can ignore this email. The link expires in 1 hour.</p>`, resetLink)

	params := &resend.SendEmailRequest{
		From:    getFrom(),
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}

	_, err := client.Emails.Send(params)
	return err
}

// SendOrderConfirmationEmail sends an order confirmation to the given address.
// If client is nil (Resend not configured), returns nil without error so checkout does not fail.
func SendOrderConfirmationEmail(to, orderID string, totalAmount float64, itemsSummary string) error {
	client := getClient()
	if client == nil {
		return nil
	}
	subject := fmt.Sprintf("Order confirmation – #%s", orderID)
	html := fmt.Sprintf(`<p>Thank you for your order.</p>
<p><strong>Order ID:</strong> %s</p>
<p><strong>Total:</strong> KES %.2f</p>
<p><strong>Items:</strong></p>
<pre>%s</pre>
<p>We will notify you when your order ships.</p>`, orderID, totalAmount, itemsSummary)
	params := &resend.SendEmailRequest{
		From:    getFrom(),
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}
	_, err := client.Emails.Send(params)
	return err
}
