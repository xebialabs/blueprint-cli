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

var askToSaveToConfig = func(surveyOpts ...survey.AskOpt) (bool, error) {
	answer := false
	var err error
	if !SkipPrompts {
		err = survey.AskOne(
			&survey.Confirm{
				Message: "Do you want to save modify your xebialabs/config.yaml to point to new XL Release and Deploy instances",
				Default: true,
				Help:    "Your xebialabs config file stores the credentials of your XL Release and Deploy instances which the CLI uses to connect",
			},
			&answer,
			survey.Required,
			surveyOpts...,
		)
	}
	return answer, err
}
