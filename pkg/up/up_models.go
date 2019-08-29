package up

type UpParams struct {
	localMode         string
	quickSetup        bool
	advancedSetup     bool
	blueprintTemplate string
	answerFile        string
	cfgOverridden     bool
	noCleanup         bool
	destroy           bool
}
