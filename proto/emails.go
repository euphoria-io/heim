package proto

import "euphoria.io/heim/proto/emails"

const (
	PasswordChanged       = emails.Template("password-changed")
	PasswordReset         = emails.Template("password-reset")
	RoomInvitation        = emails.Template("room-invitation")
	RoomInvitationWelcome = emails.Template("room-invitation-welcome")
	WelcomeEmail          = emails.Template("welcome")
)

var (
	EmailCommonData = commonData{
		"EmailFromAddress": "noreply@euphoria.io",
		"ReplyToAddress":   "help@euphoria.io",
		"SiteName":         "heim",
		"SiteURL":          "https://heim.invalid",
	}

	EmailScenarios = map[emails.Template]map[string]emails.TemplateTest{
		WelcomeEmail: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: EmailCommonData.And(map[string]interface{}{
					"VerifyEmailURL":      "https://test.invalid/verify",
					"ContactEmailAddress": "help@test.invalid",
					"AccountEmailAddress": "user@somewhere.invalid",
				}),
			},
		},

		PasswordChanged: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: EmailCommonData.And(map[string]interface{}{"AccountName": "yourname"}),
			},
		},

		PasswordReset: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: EmailCommonData.And(map[string]interface{}{
					"AccountName":      "yourname",
					"PasswordResetURL": "https://test.invalid/reset",
				}),
			},
		},

		RoomInvitation: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: EmailCommonData.And(map[string]interface{}{
					"SenderName":    "thatguy",
					"RoomName":      "butts",
					"RoomURL":       "https://heim.invalid/room/butts",
					"SenderMessage": "hey, i heard you like butts",
				}),
			},
		},

		RoomInvitationWelcome: map[string]emails.TemplateTest{
			"default": emails.TemplateTest{
				Data: EmailCommonData.And(map[string]interface{}{
					"SenderName":    "thatguy",
					"RoomName":      "cabal",
					"RoomPrivacy":   "private",
					"RoomURL":       "https://heim.invalid/room/cabal",
					"SenderMessage": "let's move our machinations here",
				}),
			},
		},
	}
)

type commonData map[string]interface{}

func (cd commonData) And(data map[string]interface{}) map[string]interface{} {
	for key, val := range cd {
		if _, ok := data[key]; !ok {
			data[key] = val
		}
	}
	return data
}

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
