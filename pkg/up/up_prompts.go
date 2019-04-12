package up

import "gopkg.in/AlecAivazis/survey.v1"

func askSetupMode(surveyOpts ...survey.AskOpt) string {
	answer := ""
	survey.AskOne(
		&survey.Select{
			Message: "Select the setup mode?",
			Options: []string{"advanced", "quick"},
			Default: "advanced",
		},
		&answer,
		survey.Required,
		surveyOpts...,
	)
	return answer
}
