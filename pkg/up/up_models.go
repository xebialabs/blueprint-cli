package up

type UpParams struct {
	LocalMode         string
	QuickSetup        bool
	AdvancedSetup     bool
	BlueprintTemplate string
	AnswerFile        string
	CfgOverridden     bool
	NoCleanup         bool
	Destroy           bool
}
