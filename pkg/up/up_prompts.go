package up

import (
	"gopkg.in/AlecAivazis/survey.v1"
)

func askSetupMode(surveyOpts ...survey.AskOpt) (string, error) {
	answer := ""
	var err error

	if !SkipPrompts {
		err = survey.AskOne(
			&survey.Select{
				Message: "Select the setup mode?",
				Options: []string{"advanced", "quick"},
				Default: "advanced",
				Help:    "Quick setup will use sensible default values for many of the options while Advanced setup will let the user provide values for all the options",
			},
			&answer,
			survey.Required,
			surveyOpts...,
		)
	}
	return answer, err
}

func askOverrideAnswerFile(surveyOpts ...survey.AskOpt) (bool, error) {
	answer := true
	var err error
	if !SkipPrompts {
		err = survey.AskOne(
			&survey.Confirm{
				Message: "Parameters in the existing answer file will be overridden by the answer file saved in  Kubernetes, Do you want to continue?",
				Default: true,
			},
			&answer,
			survey.Required,
			surveyOpts...,
		)
	}
	return answer, err
}
