package mailer

import (
	"fmt"

	resend "github.com/resend/resend-go/v2"
)

type Mailer struct {
	client *resend.Client
	from   string
}

func NewMailer(apiKey, from string) *Mailer {
	client := resend.NewClient(apiKey)
	return &Mailer{
		client: client,
		from:   from,
	}
}

func (m *Mailer) SendWelcomeEmail(to, firstName string) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: "¡Bienvenido a BankAPI!",
		Html: fmt.Sprintf(`
			<h1>¡Hola %s!</h1>
			<p>Tu cuenta en BankAPI ha sido creada exitosamente.</p>
			<p>Ya puedes crear cuentas bancarias, depositar dinero y hacer transferencias.</p>
			<br>
			<p>El equipo de BankAPI</p>
		`, firstName),
	}

	_, err := m.client.Emails.Send(params)
	return err
}

func (m *Mailer) SendLoginNotification(to, firstName string) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: "Nuevo inicio de sesión — BankAPI",
		Html: fmt.Sprintf(`
			<h1>Hola %s</h1>
			<p>Detectamos un nuevo inicio de sesión en tu cuenta.</p>
			<p>Si no fuiste tú, cambia tu contraseña inmediatamente.</p>
			<br>
			<p>El equipo de BankAPI</p>
		`, firstName),
	}

	_, err := m.client.Emails.Send(params)
	return err
}

func (m *Mailer) SendDepositNotification(to, firstName string, amount int64, currency string) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: "Depósito recibido — BankAPI",
		Html: fmt.Sprintf(`
			<h1>Hola %s</h1>
			<p>Has recibido un depósito en tu cuenta.</p>
			<p><strong>Monto:</strong> %d %s</p>
			<br>
			<p>El equipo de BankAPI</p>
		`, firstName, amount, currency),
	}

	_, err := m.client.Emails.Send(params)
	return err
}

func (m *Mailer) SendTransferNotification(to, firstName string, amount int64, currency string, toAccountId int) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: "Transferencia enviada — BankAPI",
		Html: fmt.Sprintf(`
			<h1>Hola %s</h1>
			<p>Tu transferencia ha sido procesada exitosamente.</p>
			<p><strong>Monto:</strong> %d %s</p>
			<p><strong>Cuenta destino:</strong> #%d</p>
			<br>
			<p>El equipo de BankAPI</p>
		`, firstName, amount, currency, toAccountId),
	}

	_, err := m.client.Emails.Send(params)
	return err
}

func (m *Mailer) SendSecurityAlert(to, firstName string) error {
	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: "⚠️ Alerta de seguridad — BankAPI",
		Html: fmt.Sprintf(`
			<h1>Hola %s</h1>
			<p><strong>Tu cuenta ha sido bloqueada temporalmente.</strong></p>
			<p>Detectamos múltiples intentos fallidos de inicio de sesión.</p>
			<p>Tu cuenta estará bloqueada por <strong>30 minutos</strong>.</p>
			<p>Si no fuiste tú, te recomendamos cambiar tu contraseña inmediatamente.</p>
			<br>
			<p>El equipo de BankAPI</p>
		`, firstName),
	}

	_, err := m.client.Emails.Send(params)
	return err
}