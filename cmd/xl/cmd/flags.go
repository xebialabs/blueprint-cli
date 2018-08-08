package cmd

import "github.com/spf13/pflag"

const (
	usageFilename                 = "a filename for processing in YAML format (e.g.: xld.yaml)"
	usageSkipOptional             = "skip asking for optional fields when true (default: false)"
	usageSrvApplicationsHomeXld   = "applications home directory for XL Deploy (default: Applications)"
	usageSrvConfigurationHomeXld  = "configuration home directory for XL Deploy (default: Configuration)"
	usageSrvXlrHome               = "home directory for XL Release (optional)"
	usageSrvContextRoot           = "application context root (optional)"
	usageSrvEnvironmentsHomeXld   = "environments home directory for XL Deploy (default: Environments)"
	usageSrvHost                  = "server hostname or IP address"
	usageSrvInfrastructureHomeXld = "infrastructure home directory for XL Deploy (default: Infrastructure)"
	usageSrvName                  = "server name, which is an identifier and must be unique"
	usageSrvPassword              = "password for server access"
	usageSrvPort                  = "server port"
	usageSrvSsl                   = "server protocol (1: http or 2: https)"
	usageSrvType                  = "server type (1: xl-deploy or 2: xl-release)"
	usageSrvUsername              = "username for server access"
	usageXldDefault               = "name of the XL Deploy server to use"
	usageXlrDefault               = "name of the XL Release server to use"
	usageXldUrl                   = "URL to access XL deploy (e.g.: http://path.to.xld.server/context-root)"
	usageXldUsername              = "username for XL deploy server"
	usageXldPassword              = "password for XL deploy server"
	usageXlrUrl                   = "URL to access XL release (e.g.: http://path.to.xlr.server/context-root)"
	usageXlrUsername              = "username for XL release server"
	usageXlrPassword              = "password for XL release server"
)

var (
	cfgFile               string
	filename              []string
	skipOptional          bool
	xld                   string
	xlr                   string
	xldUrl                string
	xldUsername           string
	xldPassword           string
	xldApplicationsHome   string
	xldConfigurationHome  string
	xldEnvironmentHome    string
	xldInfrastructureHome string
	xlrUrl                string
	xlrUsername           string
	xlrPassword           string
	xlrHome               string
)

//Server fields
var (
	srvName                  string
	srvType                  string
	srvHost                  string
	srvPort                  string
	srvUsername              string
	srvPassword              string
	srvSsl                   string
	srvContextRoot           string
	srvApplicationsHomeXld   string
	srvConfigurationHomeXld  string
	srvEnvironmentsHomeXld   string
	srvInfrastructureHomeXld string
	srvXlrHome               string
)

func setFilenameFlags(set *pflag.FlagSet) {
	set.StringArrayVarP(&filename, "filename", "f", []string{}, usageFilename)
}

//Set flags for specifying a server for the provided command (set *pflag.FlagSet).
//Default flags are: --name (-n), --type (-t), --host, --port (-p),
//--username (-u), --password, --ssl, --context
func setServerFlags(set *pflag.FlagSet) {
	set.StringVarP(&srvName, "name", "n", "", usageSrvName)
	set.StringVarP(&srvType, "type", "t", "", usageSrvType)
	set.StringVar(&srvHost, "host", "", usageSrvHost)
	set.StringVarP(&srvPort, "port", "p", "", usageSrvPort)
	set.StringVarP(&srvUsername, "username", "u", "", usageSrvUsername)
	set.StringVar(&srvPassword, "password", "", usageSrvPassword)
	set.StringVar(&srvSsl, "ssl", "", usageSrvSsl)
	set.StringVar(&srvContextRoot, "context", "", usageSrvContextRoot)
	set.StringVar(&srvApplicationsHomeXld, "xld-applications-home", "", usageSrvApplicationsHomeXld)
	set.StringVar(&srvConfigurationHomeXld, "xld-configuration-home", "", usageSrvConfigurationHomeXld)
	set.StringVar(&srvEnvironmentsHomeXld, "xld-environments-home", "", usageSrvEnvironmentsHomeXld)
	set.StringVar(&srvInfrastructureHomeXld, "xld-infrastructure-home", "", usageSrvInfrastructureHomeXld)
	set.StringVar(&srvXlrHome, "xlr-home", "", usageSrvXlrHome)
}

func setApplyFlags(set *pflag.FlagSet) {
	set.StringVar(&xld, "xld", "", usageXldDefault)
	set.StringVar(&xlr, "xlr", "", usageXlrDefault)
	set.StringVar(&xldUrl, "xld-url", "", usageXldUrl)
	set.StringVar(&xldUsername, "xld-username", "", usageXldUsername)
	set.StringVar(&xldPassword, "xld-password", "", usageXldPassword)
	set.StringVar(&xldApplicationsHome, "xld-applications-home", "", usageSrvApplicationsHomeXld)
	set.StringVar(&xldConfigurationHome, "xld-configuration-home", "", usageSrvConfigurationHomeXld)
	set.StringVar(&xldEnvironmentHome, "xld-environment-home", "", usageSrvEnvironmentsHomeXld)
	set.StringVar(&xldInfrastructureHome, "xld-infrastructure-home", "", usageSrvInfrastructureHomeXld)
	set.StringVar(&xlrUrl, "xlr-url", "", usageXlrUrl)
	set.StringVar(&xlrUsername, "xlr-username", "", usageXlrUsername)
	set.StringVar(&xlrPassword, "xlr-password", "", usageXlrPassword)
	set.StringVar(&xlrHome, "xlr-home", "", usageSrvXlrHome)
}

func setSkipOptionalFlags(set *pflag.FlagSet) {
	set.BoolVar(&skipOptional, "skip-optional", false, usageSkipOptional)
}
