package proto

import (
	"fmt"
	"html/template"
	"net/url"

	"euphoria.io/heim/proto/emails"
)

const (
	PasswordChangedEmail       = emails.Template("password-changed")
	PasswordResetEmail         = emails.Template("password-reset")
	RoomInvitationEmail        = emails.Template("room-invitation")
	RoomInvitationWelcomeEmail = emails.Template("room-invitation-welcome")
	WelcomeEmail               = emails.Template("welcome")
)

type CommonEmailParams struct {
	emails.TemplateDataCommon
	EmailDomain   string `yaml:"email_domain"`
	SiteName      string `yaml:"site_name"`
	SiteURL       string `yaml:"site_url"`
	HelpAddress   string `yaml:"help_address"`
	SenderAddress string `yaml:"sender_address"`
}

func (p *CommonEmailParams) SiteURLShort() string { return p.TemplateDataCommon.LocalDomain }

func (p *CommonEmailParams) EmailPreferencesURL() string {
	// TODO: incorporate token
	return fmt.Sprintf("%s/prefs/emails", p.SiteURL)
}

type WelcomeEmailParams struct {
	*CommonEmailParams
	VerificationToken string
}

func (p WelcomeEmailParams) Subject() string { return fmt.Sprintf("welcome to %s!", p.SiteName) }

func (p *WelcomeEmailParams) VerifyEmailURL() string {
	v := url.Values{
		"email": []string{p.AccountEmailAddress()},
		"token": []string{p.VerificationToken},
	}
	u := url.URL{
		Path:     "/prefs/verify",
		RawQuery: v.Encode(),
	}
	return p.SiteURL + u.String()
}

type PasswordChangedEmailParams struct {
	*CommonEmailParams
	AccountName string
}

func (p PasswordChangedEmailParams) Subject() string {
	return fmt.Sprintf("your %s account password has been changed", p.SiteName)
}

type PasswordResetEmailParams struct {
	*CommonEmailParams
	AccountName  string
	Confirmation string
}

func (p PasswordResetEmailParams) Subject() string {
	return fmt.Sprintf("password reset request for your %s account", p.SiteName)
}

func (p PasswordResetEmailParams) ResetPasswordURL() string {
	v := url.Values{
		"confirmation": []string{p.Confirmation},
	}
	u := url.URL{
		Path:     "/prefs/reset-password",
		RawQuery: v.Encode(),
	}
	return p.SiteURL + u.String()
}

type RoomInvitationEmailParams struct {
	*CommonEmailParams
	AccountName   string
	RoomName      string
	SenderName    string
	SenderMessage string
}

func (p RoomInvitationEmailParams) Subject() template.HTML {
	return template.HTML(fmt.Sprintf("%s invites you to join &%s", p.SenderName, p.RoomName))
}

func (p RoomInvitationEmailParams) RoomURL() string {
	return fmt.Sprintf("%s/room/%s", p.SiteURL, p.RoomName)
}

type RoomInvitationWelcomeEmailParams struct {
	*CommonEmailParams
	AccountName   string
	RoomName      string
	RoomPrivacy   string
	SenderName    string
	SenderMessage string
}

func (p RoomInvitationWelcomeEmailParams) Subject() string {
	return fmt.Sprintf("%s invites you to join a chatroom on %s", p.SenderName, p.SiteName)
}

func (p RoomInvitationWelcomeEmailParams) RoomURL() string {
	return fmt.Sprintf("%s/room/%s", p.SiteURL, p.RoomName)
}

var (
	DefaultCommonEmailParams = &CommonEmailParams{
		TemplateDataCommon: emails.TemplateDataCommon{
			LocalDomain: "heim.invalid",
		},
		SenderAddress: "noreply@heim.invalid",
		HelpAddress:   "help@heim.invalid",
		SiteName:      "heim",
		SiteURL:       "https://heim.invalid",
	}

	EmailScenarios = map[emails.Template]map[string]emails.TemplateTest{
		WelcomeEmail: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: &WelcomeEmailParams{
					CommonEmailParams: DefaultCommonEmailParams,
					VerificationToken: "token",
				},
			},
		},

		PasswordChangedEmail: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: &PasswordChangedEmailParams{
					CommonEmailParams: DefaultCommonEmailParams,
					AccountName:       "yourname",
				},
			},
		},

		PasswordResetEmail: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: &PasswordResetEmailParams{
					CommonEmailParams: DefaultCommonEmailParams,
					AccountName:       "yourname",
				},
			},
		},

		RoomInvitationEmail: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: &RoomInvitationEmailParams{
					CommonEmailParams: DefaultCommonEmailParams,
					SenderName:        "(‿|‿)",
					RoomName:          "butts",
					SenderMessage:     "hey, i heard you like butts",
				},
			},
		},

		RoomInvitationWelcomeEmail: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: &RoomInvitationWelcomeEmailParams{
					CommonEmailParams: DefaultCommonEmailParams,
					SenderName:        "thatguy",
					RoomName:          "cabal",
					RoomPrivacy:       "private",
					SenderMessage:     "let's move our machinations here",
				},
			},
		},
	}
)

func ValidateEmailTemplates(templater *emails.Templater) []error {
	errors := []error{}
	for templateName, testCases := range EmailScenarios {
		testList := make([]emails.TemplateTest, 0, len(testCases))
		for _, testCase := range testCases {
			testList = append(testList, testCase)
		}
		errors = append(errors, templater.Validate(templateName, testList...)...)
	}
	if len(errors) == 0 {
		return nil
	}
	return errors
}
