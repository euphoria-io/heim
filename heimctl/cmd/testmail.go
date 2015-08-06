package cmd

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/templates"
	"euphoria.io/scope"
)

func init() {
	register("testmail", &testmailCmd{})
}

type testmailCmd struct {
	smtpHost string
	username string
	password string
	scenario string
	to       string
}

func (testmailCmd) desc() string {
	return "send a sample email to someone"
}

func (testmailCmd) usage() string {
	return "testmail SCENARIO DESTINATION"
}

func (cmd *testmailCmd) longdesc() string {
	return fmt.Sprintf(`
	Send a sample email through an SMTP server. An email for the given
	SCENARIO will be constructed and delivered to the given DESTINATION.
	The list of scenarios are:

	%s

	The value of DESTINATION should be an email address.
`[1:],
		cmd.listScenarios())
}

func (testmailCmd) listScenarios() string {
	scenarios := []string{}
	for templateName, testCases := range proto.EmailScenarios {
		if len(testCases) == 1 {
			scenarios = append(scenarios, fmt.Sprintf("    * %s", templateName))
		} else {
			for scenarioName := range testCases {
				scenarios = append(scenarios, fmt.Sprintf("    * %s-%s", templateName, scenarioName))
			}
		}
	}
	sort.Strings(scenarios)
	return strings.Join(scenarios, "\n")
}

func (testmailCmd) resolveScenario(scenario string) (string, templates.TemplateTest, error) {
	for templateName, testCases := range proto.EmailScenarios {
		if strings.HasPrefix(scenario, string(templateName)) {
			if len(testCases) == 1 && string(templateName) == scenario {
				for _, testCase := range testCases {
					return templateName, testCase, nil
				}
			}
			for scenarioName, testCase := range testCases {
				if fmt.Sprintf("%s-%s", templateName, scenarioName) == scenario {
					return templateName, testCase, nil
				}
			}
		}
	}
	return "", templates.TemplateTest{}, fmt.Errorf("unknown scenario: %s", scenario)
}

func (cmd *testmailCmd) flags() *flag.FlagSet { return flag.NewFlagSet("testemail", flag.ExitOnError) }

func (cmd *testmailCmd) run(ctx scope.Context, args []string) error {
	if len(args) < 2 {
		fmt.Printf("Usage: %s\n\n", cmd.usage())

		fmt.Printf("Available scenarios:\n\n")
		fmt.Println(cmd.listScenarios())
		fmt.Println()
		return nil
	}

	templateName, testCase, err := cmd.resolveScenario(args[0])
	if err != nil {
		return err
	}

	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}

	emailer, err := cfg.Email.Get(cfg)
	if err != nil {
		return err
	}

	msgID, err := emailer.Send(ctx, args[1], templateName, testCase.Data)
	if err != nil {
		return fmt.Errorf("send failed: %s", err)
	}

	fmt.Printf("Sent email successfully.\nMessage ID: %s\n", msgID)
	return nil
}
