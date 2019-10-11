package up

type UpParams struct {
	LocalPath         string
	QuickSetup        bool
	AdvancedSetup     bool
	BlueprintTemplate string
	AnswerFile        string
	CfgOverridden     bool
	NoCleanup         bool
	Undeploy          bool
	DryRun            bool
	SkipK8sConnection bool
	SkipPrompts       bool
}
