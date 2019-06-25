# Blueprints

---------------

## Blueprint YAML Definition File Structure

### Root Fields

| Field Name | Expected value | Examples | Required |
|:----------: |:--------------: |:---------: |:--------: |
| **apiVersion** | `xl/v2` | — | ✔ |
| **kind** | `Blueprint` | — | ✔ |
| **metadata** | — | *see below* | **x** |
| **spec** | — | *see below* | ✔ |

#### Metadata Fields

| Field Name | Expected value | Examples | Required |
|:-----------: |:--------------: |--------------------------------------------------------- |:--------: |
| **name** | — | Sample Project | **x** |
| **description** | — | A long description for describing the blueprint project | **x** |
| **author** | — | XebiaLabs | **x** |
| **version** | — | 2.0 | **x** |
| **instructions** | — | You need to start your docker containers before applying the blueprint | **x** |

The `instructions` field will be displayed after the blueprint is generated.

#### Spec fields

The spec field holds parameters and files

##### Parameters Fields

Parameters are defined by the blueprint creator in the `blueprint.yaml` file, it can be used in the blueprint template files. If the `prompt` field is defined for a parameter in the `blueprint.yaml` file, the user will be prompted to enter its value during execution of the blueprint. By default parameter values will be used to replace variables in template files during blueprint generation. You can find all possible parameter options below.

| Field Name | Expected value(s) | Examples | Default Value | Required | Description |
|:--------------: |:--------------------: |------------------------------------------------------------ |:-------------: |:---------------------------------------: |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **name** | — | AppName | — | ✔ | Parameter name, to be used in template placeholders |
| **type** | `Input`/<br>`SecretInput`/<br>`Select`/<br>`Confirm`/<br>`Editor`/<br>`SecretEditor`/<br>`File`/<br>`SecretFile` | | — | Required when `value` is not set | Type of the prompt input(Type explanations below)<br> When type is `SecretInput`, `SecretEditor` or `SecretFile` the parameter is saved in `secrets.xlvals` files so that they won't be checked in GIT repo and will not be replaced with actual value by default in the template files|
| **prompt** | - | What is your application name? | — | Required when `value` is not set | Question to prompt. |
| **value** | — | `eu-west-1`/<br>`!expr "Foo == 'foo' ? 'A' : 'B'"` | — | **x** | If present, user will not be asked a question to provide value. |
| **default** | — | `eu-west-1`/<br>`!expr "Foo == 'foo' ? 'A' : 'B'"` | — | **x** | Default value, will be present during the question prompt. Also will be the parameter value if question is skipped. |
| **description** | — | Application name, will be used in various AWS resource names | — | **x** | If present, will be used as help text for question prompt |
| **label** | — | Application name | — | **x** | If present, will be used instead of name in summary table |
| **options** | — | `- eu-west-1`<br>`- us-east-1`<br>`- us-west-1`<br>`- label: us west 1`<br>&nbsp;&nbsp;`value: us-west-1`<br>`-!expr "Foo == 'foo' ? ('A', 'B') : ('C', 'D')"` | — | Required for `Select` input type | Set of options for the `Select` input type. Can consist of any number of text values, label/value pairs or values retrieved from an expression. |
| **validate** | `!expr` tag | `!expr "regex('[a-z]*', paramName)"`| — | **x** | Validation expression to be verified at the time of user input, any combination of expressions and expression functions can be used. <br>The current parameter name must be passed to the validation function. Expected result of the expression evaluated is of type boolean. |
| **promptIf** | — | `CreateNewCluster`/<br>`!expr "CreateNewCluster == true"` | — | **x** | If this question needs to be asked to user depending on the value of another, promptIf field can be defined.<br>A valid parameter name should be given and the parameter name used should have been defined before order-wise. Expression tags also can be used, but expected result should always be boolean. Should not be set along with `value` |
| **saveInXlvals** | `true`/`false` | — | `true` for `SecretInput`, `SecretEditor` and `SecretFile` fields<br>`false` for other fields | **x** | If true, output parameter will be included in the `values.xlvals` output file. `SecretInput`, `SecretEditor` and `SecretFile` parameters will always be written to `secrets.xlvals` file regardless of what you set for this field |
| **replaceAsIs** | `true`/`false` | — | `false` | **x** | `SecretInput`, `SecretEditor` and `SecretFile` field values are normally not directly used in Go template files, instead it will be referred using `!value ParameterName` syntax. If `replaceAsIs` is set to `true`, output parameter will be used as raw value instead of with `!value` tag in Go templates. Useful in cases where parameter will be used with a post-process function in any template file. <br/> This parameter is only valid for `SecretInput`, `SecretEditor` and `SecretFile` fields, for other fields it will produce a validation error. |
| **revealOnSummary** | `true`/`false` | — | `false` | **x** | If set to `true`, the value will be present on the summary table. <br/> This parameter is only valid for `SecretInput`, `SecretEditor` and `SecretFile` fields, for other fields it will produce a validation error. |

> Note #1: `File` type doesn't support `value` parameter. `default` parameter for this field expects to have a file path instead of final value string.

> Note #2: parameters with `SecretInput`, `SecretEditor` and `SecretFile` type supports default values as well. When a `SecretInput`, `SecretEditor` or `SecretFile` parameter question is being asked to the user, the default value will be shown on the prompt as raw text, and if the user enters an empty response for the question this default value will be used instead.
###### Types

The types that can be used for inputs are below

`Input`: Used for simple text or number inputs.

`SecretInput`: Used for simple secret or password inputs. These are by default saved in `secrets.xlvals` files so that they won't be checked in GIT repo and will not be replaced with actual value in the template files.

`Select`: Used for select inputs where user can choose from given options.

`Confirm`: Used for boolean inputs.

`Editor`: Used for multiline or complex text input.

`SecretEditor`: Used for multiline or complex secret inputs. These are by default saved in `secrets.xlvals` files so that they won't be checked in GIT repo and will not be replaced with actual value in the template files.

`File`: Used for fetching the content of a given file path.

`SecretFile`: Used for fetching the content of a given file path and treat it as secret. These are by default saved in `secrets.xlvals` files so that they won't be checked in GIT repo and will not be replaced with actual value in the template files.

##### Files Fields

| Field Name | Expected value(s) | Examples | Default Value | Required | Explanation |
|:--------------: |:--------------------: |------------------------------------------------------------ |:-------------: |:---------------------------------------: |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **path** | — | `xebialabs/xlr-pipeline.yaml` | — | ✔ | File/template path to be copied/processed  |
| **renameTo** | — | `xebialabs/xlr-pipeline-new.yaml` | — | **x** | The name to be used for output file  |
| **writeIf** | — | `CreateNewCluster`/<br>`!expr "CreateNewCluster == true"` | — | **x** | This file will be generated only when value of a parameter or function return true.<br>A valid parameter name should be given and the parameter name used should have been defined. Expression tags also can be used, but expected result should always be boolean. |

##### IncludeBefore/IncludeAfter Fields

includeBefore/includeAfter will decide if the blueprint should be composed before or after the master blueprint, this will affect the order in which the parameters will be presented to the user and order in which files are written, Entries in before/after will stack based on order of definition.

| Field Name | Expected value(s) | Examples | Default Value | Required | Explanation |
|:--------------: |:--------------------: |------------------------------------------------------------ |:-------------: |:---------------------------------------: |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **blueprint** | — | aws/monolith | — | ✔ | The full path of the blueprint to be composed, will be looked up from the current repository being used |
| **includeIf** | — | `CreateNewCluster`/<br>`!expr "CreateNewCluster == true"` | — | **x** | This blueprint will be included only when value of a parameter or expression returns true.<br>A valid parameter name should be given and the parameter name used should have been defined. Expression tags can also be used if the returned value is a boolean. |
| **parameterOverrides** | Parameter definition | - | — | **x** | Overrides fields of the parameters defined on the blueprint included. This way we can force to skip any question by providing a value for it or by overriding its `promptIf`. Can override everything except `name` and `type` fields |
| **fileOverrides** | File definition | - | — | **x** | Can be used to override fields of any file definition in the blueprint being composed. This way we can force to skip any file by overriding its `writeIf` or rename a file by providing `renameTo`. Can override everything except `path` field |

An example `blueprint.yaml` using include for composing multiple blueprints

```yaml
apiVersion: xl/v2
kind: Blueprint
metadata:
  name: Composed blueprint
  version: 2.0
spec:
  parameters:
  - name: Foo
    prompt: what is value for Foo?

  files:
  - path: xlr-pipeline.yml
    writeIf: !expr "Foo == 'foo'"

  includeBefore: # the `aws/datalake` will be executed first followed by the current blueprint.yaml
  # we will look for `aws/datalake` in the current-repository being used
  - blueprint: aws/datalake
    # with 'parameterOverrides' we can provide values for any parameter in the blueprint being composed. This way we can force to skip any question by providing a value for it
    parameterOverrides:
    # we are overriding the value and promptIf fields of the TestFoo parameter in the `aws/datalake` blueprint
    - name: TestFoo
      value: hello
      promptIf: !expr "3 > 2"
    # 'fileOverrides' can be used to skip files and can be conditional using dependsOn
    fileOverrides:
    - path: xld-environment.yml.tmpl
      writeIf: !expr "false" # we are skipping this file
    - path: xlr-pipeline.yml
      renameTo: xlr-pipeline-new.yml # we are renaming this file since the current blueprint.yaml already has this file defined in the file section above
  includeAfter: # the `k8s/environment` will be executed after the current blueprint.yaml
  # we will look for `k8s/environment` in the current-repository being used
  - blueprint: k8s/environment
    parameterOverrides:
    - name: Test
      value: hello2
    fileOverrides:
    - path: xld-environment.yml.tmpl
      writeIf: !expr "false"

```

---------------


## Supported Custom YAML Tags

### Expression tag (`!expr`)

Blueprints support custom expressions to be used within parameter definitions, file declarations & includeBefore/includeAfter (`spec` part in YAML file). Expression tag can be used in parameter/parameterOverrides fields `default`, `value`, `promptIf`, `options`, `validate`, file/fileOverrides field `writeIf` & includeBefore/includeAfter fields `includeIf`. 

You can use a parameter defined in the parameters section inside an expression. Parameter names are case sensitive and you should define the parameter before it is used in an expression, in other words you can't refer to a parameter that will be defined after the expression is defined in the `blueprint.yaml` file or in an included blueprint.

Custom expression syntax: `!expr "EXPRESSION"`

#### Operators and types supported

* Modifiers: `+` `-` `/` `*` `&` `|` `^` `**` `%` `>>` `<<`
* Comparators: `>` `>=` `<` `<=` `==` `!=` `=~` `!~`
* Logical ops: `||` `&&`
* Numeric constants, as 64-bit floating point (`12345.678`)
* String constants (single quotes: `'foobar'`)
* Date constants (single quotes, using any permutation of RFC3339, ISO8601, ruby date, or unix date; date parsing is automatically tried with any string constant)
* Boolean constants: `true` `false`
* Parenthesis to control order of evaluation `(` `)`
* Arrays (anything separated by `,` within parenthesis: `(1, 2, 'foo')`)
* Prefixes: `!` `-` `~`
* Ternary conditional: `?` `:`
* Null coalescence: `??`

See [MANUAL.md](https://github.com/Knetic/govaluate/blob/master/MANUAL.md) from [govaluate](https://github.com/Knetic/govaluate) for more details on what types each operator supports.

#### Types

Only supported types are; `float64`, `bool`, `string`, and `arrays`. When using expressions to return values for `options`, please ensure the expression returns an array. When using expressions on `promptIf`, `writeIf` and `includeIf` fields, ensure that it returns boolean

#### Escaping characters

Sometimes you'll have parameters that have spaces, slashes, pluses, ampersands or some other character
that may be interpreted as something special. For example, the following expression will not
act as one might expect:

	"response-time < 100"

As written, it will be parsed it as "[response] minus [time] is less than 100". In reality,
"response-time" is meant to be one variable that just happens to have a dash in it.

There are two ways to work around this. First, you can escape the entire parameter name:

 	"[response-time] < 100"

Or you can use backslashes to escape only the minus sign.

	"response\\-time < 100"

Backslashes can be used anywhere in an expression to escape the very next character. Square bracketed parameter names can be used instead of plain parameter names at any time.


#### Available custom functions for expressions:

You can use the provided functions in an expression

| Function | Parameters | Examples | Description |
|:------: |:-----------: |:----------------------------------------: |:----------------: 
| **strlen** | Parameter or Text(string) | - `!expr "strlen('Foo') > 5"`<br>- `!expr "strlen(FooParameter) > 5"` | Get the length of the given string variable |
| **max** | Parameter or numbers(float64, float64) | - `!expr "max(5, 10) > 5"`<br>- `!expr "max(FooParameter, 100)"` | Get the maximum of the two given numbers |
| **min** | Parameter or numbers(float64, float64) | - `!expr "min(5, 10) > 5"`<br>- `!expr "min(FooParameter, 100)"` | Get the minimum of the two given numbers |
| **ceil** | Parameter or number(float64) | - `!expr "ceil(5.8) > 5"`<br>- `!expr "ceil(FooParameter) > 5"` | Ceil the given number to nearest whole number |
| **floor** | Parameter or number(float64) | - `!expr "floor(5.8) > 5"`<br>- `!expr "floor(FooParameter) > 5"` | Floor the given number to nearest whole number |
| **round** | Parameter or number(float64) | - `!expr "round(5.8) > 5"`<br>- `!expr "round(FooParameter) > 5"` | Round the given number to nearest whole number |
| **randPassword** | String | - `!expr "randPassword()"`| Generates a 16-character random password |
| **string** | Parameter or number(float64) | - `!expr "string(103.4)"`| Converts variable or number to string |
| **regex** | - Pattern text</br>- Value to test | - `!expr "regex('[a-zA-Z-]*', ParameterName)"`| Tests given value with the provided regular expression pattern. Return `true` or `false`. Note that `\` needs to be escaped as `\\\\` in the patterns used. |
| **isFile** | File path string | - `!expr "isFile('/test/dir/file.txt')"`| Checks if the file exists or not |
| **isDir** | Directory path string | - `!expr "isDir('/test/dir')"`| Checks if the directory exists or not |
| **isValidUrl** | URL text | - `!expr "isValidUrl('http://xebialabs.com/')"`| Checks if the given URL text is a valid URL or not. Doesn't check for the status code or availibity of the URL, just checks the structure |
| **awsCredentials** | Attribute text:</br>- `IsAvailable`</br>- `AccessKeyID`</br>- `SecretAccessKey`</br>- `ProviderName`</br> | - `!expr "awsCredentials('IsAvailable')"`| System-wide defined AWS credentials can be accessed with this function. `IsAvailable` attribute returns `true` or `false` based on if the AWS configuration file can be found in the system or not. Rest of the attributes return the text value read from AWS configuration file. `AWS_PROFILE` env variable can be set to change the active AWS profile system wide. |
| **awsRegions** | - AWS service name</br>- Index of the result list [**optional**] | - `!expr "awsRegions('ecs', 2)"`| Returns list of AWS regions that is available for the given AWS service. If the second parameter is not provided, function will return the whole list. |
| **k8sConfig** | - K8s Config attribute name(`ClusterServer`/<br>`ClusterCertificateAuthorityData`/<br>`ClusterInsecureSkipTLSVerify`/<br>`ContextCluster`/<br>`ContextNamespace`/<br>`ContextUser`/<br>`UserClientCertificateData`/<br>`UserClientKeyData`/<br>`IsAvailable`)</br>- Context name [**optional**] | - `!expr "k8sConfig('IsAvailable')"`</br>- `!expr "k8sConfig('ClusterServer', 'myContext')"` | Returns k8s config attribute value from the config file read from the system. For `IsAvailable` attribute, `true` or `false` value will be returned. If context name is not defined, `current-contex` will be read from the config file. |

An example `blueprint.yaml` using expressions for complex behaviors

```yaml
apiVersion: xl/v2
kind: Blueprint
metadata:
  name: Blueprint Project
  description: A Blueprint project
  author: XebiaLabs
  version: 1.0
spec:
  parameters:
  - name: Provider
    prompt: what is your Kubernetes provider?
    type: Select
    options:
      - AWS
      - GCP
      - Azure
    default: AWS

  - name: Service
    prompt: What service do you want to deploy?
    type: Select
    options:
      - !expr "Provider == 'GCP' ? ('GKE', 'CloudStorage') : (Provider == 'AWS' ? ('EKS', 'S3') : ('AKS', 'AzureStorage'))"
    default: !expr "Provider == 'GCP' ? 'GKE' : (Provider == 'AWS' ? 'EKS' : 'AKS')"

  - name: K8sClusterName
    prompt: What is your Kubernetes cluster name
    type: Input
    promptIf: !expr "Service == 'GKE' || Service == 'EKS' || Service == 'AKS'"
    default: !expr "k8sConfig('ClusterServer')"

  # AWS specific variables
  - name: UseAWSCredentialsFromSystem
    prompt: Do you want to use AWS credentials from your ~/.aws/credentials file?
    type: Confirm
    promptIf: !expr "Provider == 'AWS' && awsCredentials('IsAvailable')"

  - name: AWSAccessKey
    type: SecretInput
    prompt: What is the AWS Access Key ID?
    promptIf: !expr "Provider == 'AWS' && !UseAWSCredentialsFromSystem"
    default: !expr "awsCredentials('AccessKeyID')"

  - name: AWSAccessSecret
    prompt: What is the AWS Secret Access Key?
    type: SecretInput
    promptIf: !expr "Provider == 'AWS' && !UseAWSCredentialsFromSystem"
    default: !expr "awsCredentials('SecretAccessKey')"

  - name: AWSRegion
    type: Select
    prompt: "Select the AWS region:"
    promptIf: !expr "Provider == 'AWS'"
    options:
      - !expr "awsRegions('ecs')"
    default: !expr "awsRegions('ecs', 0)"

  files:
  - path: xld-k8s-infrastructure.yml
    writeIf: !expr "Service == 'GKE' || Service == 'EKS' || Service == 'AKS'"
  - path: xld-storage-infrastructure.yml
    writeIf: !expr "Service == 'CloudStorage' || Service == 'S3' || Service == 'AzureStorage'"
```

---------------

## Go Templates

In blueprint template files using `.tmpl` extension, GoLang templating can be used. 
Please refer to the following [cheatsheet](https://curtisvermeeren.github.io/2017/09/14/Golang-Templates-Cheatsheet) for more details how to use GoLang templates. 
Also support for additional [Sprig](http://masterminds.github.io/sprig/) functions are included in the templating engine, as well as list of custom XL functions. 
Please refer to below table for additional functions available.

| Function | Example | Description |
|:---------: |:----------------------: |:-------------------------------------------------: |
| kebabcase | `.AppName | kebabcase` | Convert string to use kebab case (separated by -) |


Note: Parameters marked as `secret` cannot be used with Go template functions & Sprig Functions since their values will not be directly replaced in the templates.

---------------

## Blueprint Repository

Remote blueprint repositories are supported for fetching blueprint files. By default, running `xl` command for the first time will generate default configuration file in your home directory (ex. `~/.xebialabs/config.yaml`), and default [XebiaLabs blueprint repository URL](https://dist.xebialabs.com/public/blueprints/) will be present in that configuration file. XL-CLI configuration file can be updated manually or appropriate command line flags can be also passed when running the command in order to specify a different remote blueprint repository. Please refer to XL-CLI documentation for detailed configuration and command line flag usage.

Example `config.yaml` blueprint configuration:

```yaml
blueprint:
  current-repository: xebialabs-github
  repositories:
    - name: xebialabs-github
      type: github
      repo-name: blueprints
      owner: xebialabs
      token: my-github-token
      branch: master
    - name: xebialabs-dist
      type: http
      url: http://dist.xebialabs.com/public/blueprints
    - name: test
      type: local
      path: /path/to/local/test/blueprints/
      ignored-dirs: .git, .vscode
      ignored-files: .DS_Store, .gitignore
```

It is possible to define multiple blueprint repositories with same or different types at the same time, but only one of them will be active at a given time. Active blueprint repository should be stated using `current-repository` field in the configuration file. When there's no defined blueprint repository, or `current-repository` field is not stated, `xl` command will auto update the config with the default XebiaLabs blueprint repository.

### Using Existing Blueprint Repositories

#### GitHub Repository Type - `type: github`

| Config Field | Expected Value | Default Value | Required | Explanation |
|:------------:|:--------------:|:-------------:| :------: | :---------: |
| name | — | — | ✔ | Repository configuration name |
| type | `github` | — | ✔ | Repository type |
| repo-name | — | — | ✔ | GitHub remote repository name |
| owner | — | — | ✔ | GitHub remote repository owner<br/>Can be different than the user accessing it |
| branch | — | `master` | **x** | GitHub remote repository branch to use |
| token | — | | **x** | GitHub user token, please refer to [GitHub documentation](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line) for generating one<br/>Repo read permission is required when generating token for XL-CLI |

> Note: When `token` field is not specified, GitHub API will be accessed in *unauthenticated* mode and rate limit will be much less than the *authenticated* mode. According to the [GitHub API documentation](https://developer.github.com/v3/#rate-limiting), *unauthenticated* rate limit per hour and per IP address is **60**, whereas *authenticated* rate limit per hour and per user is **5000**. `token` field is advised to be set in configuration for not getting any GitHub API related rate limit errors.

#### HTTP Repository Type - `type: http`

| Config Field | Expected Value | Default Value | Required | Explanation |
|:------------:|:--------------:|:-------------:| :------: | :---------: |
| name | — | — | ✔ | Repository configuration name |
| type | `http` | — | ✔ | Repository type |
| url | — | — | ✔ | HTTP repository URL, including protocol |
| username | — | | **x** | Basic authentication username |
| password | — | | **x** | Basic authentication password |

> Note: Only *basic authentication* is supported at the moment for remote HTTP repositories.

#### Local Repository Type - `type: local`

Mainly intended to be used for local development and tests. Any local path can be used as a blueprint repository with this type.

| Config Field | Expected Value | Default Value | Required | Explanation |
|:------------:|:--------------:|:-------------:| :------: | :---------: |
| name | — | — | ✔ | Repository configuration name |
| type | `local` | — | ✔ | Repository type |
| path | — | — | ✔ | Full local path where blueprint definitions are stored. `~` can be used for stating current user's home directory under Unix systems. |
| ignored-dirs | — | | **x** | List of directories, comma separated, to be ignored while traversing local path.</br>Ex. `.git, some-other-dir` |
| ignored-files | — | | **x** | List of files, comma separated, to be ignored while traversing local path.</br>Ex. `.DS_Store, .gitignore` |

> Note: Please note that in case of local repository being a too generic path, like `~`, traversing file path will be quite big and it may result in blueprint command to run very slow.

> Note: In development you can use the `-l` flag to use a local repo directly without defining it in configuration. For example to execute a blueprint in a local directory `~/mySpace/myBlueprint` you can run `xl blueprint -l ~/mySpace -b myBlueprint`.

### Creating a New Blueprint Repository

#### New GitHub Repository

Any public or private GitHub repository can be used as a remote blueprint repository. No additional setup is required on the repository. When XL-CLI configuration is directed to the repository, it will scan all folders within the repo and list available blueprints.

#### New HTTP Repository

When setting up a new HTTP blueprint repository, the most important part not to forget is to keep an up-to-date `index.json` file on the root of the repository. Since HTTP doesn't natively support directory listing, `index.json` file is used to get available blueprint information from the repository. For automatically generating a `index.json` file on your release pipeline, you can refer to the sample `generate_index.py` python script in the official [XebiaLabs blueprint GitHub repository](https://github.com/xebialabs/blueprints/blob/development/generate_index.py).

Sample `index.json` file from official XebiaLabs HTTP blueprint repository:

```json
[
"aws/monolith",
"aws/microservice-ecommerce",
"aws/datalake",
"docker/simple-demo-app"
]
```

> Note: Only *basic authentication* is supported at the moment for remote HTTP repositories.

---------------

## Blueprint Command Flags & Options

Flags and options that can be set to `xl blueprint` command are the following:

### Global Flags

- `--blueprint-current-repository` : Can be used for overriding `current-repository` field of blueprint configuration.

### Command Options

| Option (short) | Option (long) | Default Value | Examples | Explanation |
|:--------------:|:-------------:|:-------------:| :------: | :---------: |
| `-h` | `--help` | — | `xl blueprint -h` | Prints out help text for blueprint command |
| `-a` | `--answers` | — | `xl blueprint -a /path/to/answers.yaml` | When provided, values within answers file will be used as parameter input. By default strict mode is off so any value that is not provided in the file will be asked to user. |
| `-s` | `--strict-answers` | `false` | `xl blueprint -sa /path/to/answers.yaml` | If flag is set, all parameters will be requested from the answers file, and error will be thrown if one of them is not there.<br/>If not set, existing answer values will be used from answers file, and remaining ones will be asked to user from command line. |
| `-b` | `--blueprint` | | `xl blueprint -b aws/monolith`  | Looks  for the path relative to the current repository and instead of asking user which blueprint to use, it will directly fetch the specified blueprint from repository, or give an error if blueprint not found in repository |
| `-l` | `--local-repo` | | `xl blueprint -l ./templates/test -b my-blueprint`  | Local repository directory to use (bypasses active repository). Can be used along with `-b` flag to execute blueprints from your local filesystem without defining a repository for it. |
| `-d` | `--use-defaults` | | `xl blueprint -d`  | If flag is set, default fields in parameter definitions will be used as value fields, thus user will not be asked question for a parameter if a default value is present |

---------------

## Blueprint Answers File

This feature can be useful when testing blueprints or when there are too many blueprint questions to answer through command line. Command line flags `-a` and `-s`, as described above, can be given to use this feature. Input answers file format is expected to be YAML. Here's an example `answers.yaml` file:

```yaml
AppName: TestApp
ClientCert: |
    FshYmQzRUNbYTA4Icc3V7JEgLXMNjcSLY9L1H4XQD79coMBRbbJFtOsp0Yk2btCKCAYLio0S8Jw85W5mgpLkasvCrXO5
    QJGxFvtQc2tHGLj0kNzM9KyAqbUJRe1l40TqfMdscEaWJimtd4oygqVc6y7zW1Wuj1EcDUvMD8qK8FEWfQgm5ilBIldQ
ProvisionCluster: true
AWSAccessKey: accesskey
AWSAccessSecret: accesssecret
DiskSize: 100.0
```

Using answers file with `--strict-answers` flag, any command line input can be bypassed and blueprint tests can be fully automated. For more information on how to automate tests for blueprints with answers file and test case files, please refer to **Blueprint Testing** section of `blueprints` [XebiaLabs Blueprints](https://github.com/xebialabs/blueprints/blob/qpi-travis/README.md).

When answers file is provided, it will be used in the same order as the command line input. As usual, while preparing a value for the parameter following steps will be followed:

- If `promptIf` field exist, they are evaluated and based on the boolean result whether to continue or not is decided
- If `value` field is present in parameter definiton, regardless of answers file value, `value` field value is going to be used
- If answers file is present and parameter value is found within, it will be used
- If none of the above is present and the parameter is not skipped on condition, user will be asked for input through command line when `--strict-answers` is not enabled.
