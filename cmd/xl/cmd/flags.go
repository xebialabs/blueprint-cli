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
)

var (
	cfgFile      string
	filename     []string
	skipOptional bool
	url          string
	xld          string
	xlr          string
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

func setServerNameFlags(set *pflag.FlagSet) {
	set.StringVar(&xld, "xld", "default", usageXldDefault)
	set.StringVar(&xlr, "xlr", "default", usageXlrDefault)
}

func setSkipOptionalFlags(set *pflag.FlagSet) {
	set.BoolVar(&skipOptional, "skip-optional", false, usageSkipOptional)
}
