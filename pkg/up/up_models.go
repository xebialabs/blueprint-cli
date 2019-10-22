package up

type UpParams struct {
	LocalPath         string
	QuickSetup        bool
	AdvancedSetup     bool
	BlueprintTemplate string
	AnswerFile        string
	CfgOverridden     bool
	GITBranch         string
	NoCleanup         bool
	Undeploy          bool
	DryRun            bool
	SkipK8sConnection bool
	SkipPrompts       bool
	SeedVersion       string
	XLDVersions       string
	XLRVersions       string
}
