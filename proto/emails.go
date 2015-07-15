package proto

import "euphoria.io/heim/proto/emails"

const (
	WelcomeEmail = emails.Template("welcome")
)

var EmailScenarios = map[emails.Template]map[string]emails.TemplateTest{
	WelcomeEmail: map[string]emails.TemplateTest{
		"default": emails.TemplateTest{
			Data: map[string]interface{}{
				"EmailFromAddress":    "noreply@euphoria.io",
				"EmailReplyToAddress": "help@euphoria.io",
				"SiteName":            "Heim",
				"SiteURL":             "https://heim.invalid",
				"VerifyEmailURL":      "https://test.invalid/verify",
				"ContactEmailAddress": "help@test.invalid",
				"AccountEmailAddress": "user@somewhere.invalid",
			},
		},
	},
}
