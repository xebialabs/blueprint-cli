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

func askOverrideAnswerFile(surveyOpts ...survey.AskOpt) bool {
	answer := true
	survey.AskOne(
		&survey.Confirm{
			Message: "Parameters in the existing answer file will be overridden by the answer file saved in  Kubernetes, Do you want to continue?",
			Default: true,
		},
		&answer,
		survey.Required,
		surveyOpts...,
	)
	return answer
}
