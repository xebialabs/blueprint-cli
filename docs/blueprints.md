# Blueprints

---------------

## Blueprint YAML Definition File Structure

### Root Fields

| Field Name | Expected value | Examples | Required |
|:----------: |:--------------: |:---------: |:--------: |
| **apiVersion** | `xl/v1` | — | ✔ |
| **kind** | `Blueprint` | — | ✔ |
| **metadata** | — | *see below* | **x** |
| **spec** | — | *see below* | ✔ |

#### Metadata Fields

| Field Name | Expected value | Examples | Required |
|:-----------: |:--------------: |--------------------------------------------------------- |:--------: |
| **projectName** | — | Sample Project | **x** |
| **description** | — | A long description for describing the blueprint project | **x** |
| **author** | — | XebiaLabs | **x** |
| **version** | — | 1.0 | **x** |

#### Spec fields

The spec field holds parameters and files

##### Parameters Fields

Parameters are defined by the blueprint creator in the `blueprint.yaml` file, it can be used in the blueprint template files. If no value is defined for a parameter in the `blueprint.yaml` file, the user will be prompted to enter its value during execution of the bluerpint. By default parameter values will be used to replace variables in template files during blueprint generation. You can find all possible parameter options below.

| Field Name | Expected value(s) | Examples | Default Value | Required | Explanation |
|:--------------: |:--------------------: |------------------------------------------------------------ |:-------------: |:---------------------------------------: |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **name** | — | AppName | — | ✔ | Variable name, to be used in templates |
| **type** | `Input`/`Select`/`Confirm`/`Editor`/`File` | | — | ✔ | Type of the prompt input |
| **value** | — | - eu-west-1<br>-`!fn aws.regions(ecs)[0]`<br>-`!expression "Foo == 'foo' ? 'A' : 'B'"` | — | **x** | If present, user will not be asked a question to provide value. |
| **default** | — | - eu-west-1<br>-`!fn aws.regions(ecs)[0]`<br>-`!expression "Foo == 'foo' ? 'A' : 'B'"` | — | **x** | Default value, will be present during the question prompt. Also will be the variable value if question is skipped. |
| **description** | — | Application name, will be used in various AWS resource names | — | **x** | If present, will be used instead of default question text |
| **secret** | `true`/`false` | — | `false` | **x** | Variables that are marked as secret are saved in `secrets.xlvals` files so that they won't be checked in GIT repo and will not be replaced by default in the template files |
| **options** | — | - eu-west-1<br>- us-east-1<br>- us-west-1<br>-`!fn aws.regions(ecs)`<br>-`!expression "Foo == 'foo' ? ('A', 'B') : ('C', 'D')"` | — | required for `Select` input type | Set of options for the `Select` input type. Can consist of any number of text values or custom tags |
| **pattern** | — | `[a-z-]* `| — | **x** | Validation regular expression, to be verified at the time of user input |
| **dependsOnTrue** | — | - CreateNewCluster<br>- `!fn aws.credentials().IsAvailable`<br>- `!expression "CreateNewCluster == true"` | — | **x** | If this question is need to be asked to user depending on the value of another, dependsOn field can be defined.<br>A valid variable name should be given and the variable name used should have been defined before order-wise. Function tags also can be used, but expected result should always be boolean. |
| **dependsOnFalse** | — | - CreateNewCluster<br>- `!fn aws.credentials().IsAvailable`<br>- `!expression "CreateNewCluster == true"` | — | **x** | Reverse logic for dependsOnTrue, see above
| **saveInXlVals** | `true`/`false` | — | `true` for secret fields<br>`false` for other fields | **x** | If true, output variable will be included in the `values.xlvals` output file. By default every secret field will be written to `secrets.xlvals` file and this setting doesn't effect that functionality |

> Note #1: `File` type doesn't support `value` parameter. `default` parameter for this field expects to have a file path instead of final value string.

> Note #2: parameters marked as `secret` supports default values as well. When a `secret` parameter question is being asked to the user, the default value will be shown on the prompt as raw text, and if the user enters an empty response for the question this default value will be used instead.

##### Files Fields

| Field Name | Expected value(s) | Examples | Default Value | Required | Explanation |
|:--------------: |:--------------------: |------------------------------------------------------------ |:-------------: |:---------------------------------------: |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **path** | — | `xebialabs/xlr-pipeline.yaml` | — | ✔ | File/template path to be copied/processed  |
| **dependsOnTrue** | — | - CreateNewCluster<br>- `!fn aws.credentials().IsAvailable`<br>- `!expression "CreateNewCluster == true"` | — | **x** | This file will be generated only when value of a variable or function return true.<br>A valid variable name should be given and the variable name used should have been defined. Function tags also can be used, but expected result should always be boolean. |
| **dependsOnFalse** | — | - CreateNewCluster<br>- `!fn aws.credentials().IsAvailable`<br>- `!expression "CreateNewCluster == true"` | — | **x** | This file will be generated only when value of a variable or function return false.<br>A valid variable name should be given and the variable name used should have been defined. Function tags also can be used, but expected result should always be boolean.|

---------------

## Supported Custom YAML Tags

### Function tag (`!fn`)

Blueprints support custom functions to be used within variable definitions and file declarations (`spec` part in YAML file). Function tag can be used in `value`, `default`, `options`, `dependsOnTrue` and `dependsOnFalse` fields.

Custom function syntax: `!fn DOMAIN.MODULE (PARAMETERS...).ATTRIBUTE|[INDEX]`

#### Available custom functions:

| Domain | Module | Examples | Parameters | Attributes/Index | Description |
|:------: |:-----------: |:----------------------------------------: |:----------------: |------------------------------------------------------- |:------------------------------------------------------------: |
| **aws** | **credentials** | `aws.credentials().AccessKeyID` | **[optional]** Profile name | - AccessKeyID<br>- SecretAccessKey<br>- SessionToken<br>- ProviderName<br>- IsAvailable | Read AWS credentials package from system aws-cli config file |
| **aws** | **regions** | `aws.regions(ecs)`<br>`aws.regions(ecs)[0]` | AWS service ID | Any index of the resulting array | Get list of available regions for the specified AWS service |
| **k8s** | **config** | `k8s.config().IsAvailable`<br>`k8s.config(myContext).ClusterServer` | **[optional]** Context name | - ClusterServer<br>- ClusterCertificateAuthorityData<br>- ClusterInsecureSkipTLSVerify<br>- ContextCluster<br>- ContextNamespace<br>- ContextUser<br>- UserClientCertificateData<br>- UserClientKeyData<br>- IsAvailable | Get the Kubernetes context specified. If no context is specified the current context in use will be fetched. the base64 encoded values are decoded automatically|

### Expression tag (`!expression`)

Blueprints support custom expressions to be used within variable definitions and file declarations (`spec` part in YAML file). Expression tag can be used in `value`, `default`, `options`, `dependsOnTrue` and `dependsOnFalse` fields. 

You can use a variable defined in the parameter section inside an expression. Variable names are case sensitive and you should define the variable before it is used in an expression, in other words you can't refer to a variable that will be defined after the expression in defined in the `blueprint.yaml` file.

Custom expression syntax: `!expression "EXPRESSION"`

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

Only supported types are; `float64`, `bool`, `string`, and `arrays`. When using expressions to return values for `options`, please ensure the expression returns an array. When using expressions on `dependsOnTrue` and `dependsOnFalse` fields, ensure that it returns boolean

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
|:------: |:-----------: |:----------------------------------------: |:----------------: |------------------------------------------------------- |:------------------------------------------------------------: |
| **strlen** | Variable or Text(string) | - `!expression "strlen('Foo') > 5"`<br>- `!expression "strlen(FooVariable) > 5"` | Get the length of the given string variable |
| **max** | Variable or numbers(float64, float64) | - `!expression "max(5, 10) > 5"`<br>- `!expression "max(FooVariable, 100)"` | Get the maximum of the two given numbers |
| **min** | Variable or numbers(float64, float64) | - `!expression "min(5, 10) > 5"`<br>- `!expression "min(FooVariable, 100)"` | Get the minimum of the two given numbers |
| **ceil** | Variable or number(float64) | - `!expression "ceil(5.8) > 5"`<br>- `!expression "ceil(FooVariable) > 5"` | Ceil the given number to nearest whole number |
| **floor** | Variable or number(float64) | - `!expression "floor(5.8) > 5"`<br>- `!expression "floor(FooVariable) > 5"` | Floor the given number to nearest whole number |
| **round** | Variable or number(float64) | - `!expression "round(5.8) > 5"`<br>- `!expression "round(FooVariable) > 5"` | Round the given number to nearest whole number |
| **randPassword** | String | - `!expression "randPassword()"`| Generates a 16-character random password |

An example `blueprint.yaml` using expressions for complex behaviors

```
apiVersion: xl/v1
kind: Blueprint
metadata:
  projectName: Blueprint Project
  description: A Blueprint project
  author: XebiaLabs
  version: 1.0
spec:
  parameters:
  - name: Provider
    description: what is your Kubernetes provider?
    type: Select
    options:
      - AWS
      - GCP
      - Azure
    default: AWS

  - name: Service
    description: What service do you want to deploy?
    type: Select
    options:
      - !expression "Provider == 'GCP' ? ('GKE', 'CloudStorage') : (Provider == 'AWS' ? ('EKS', 'S3') : ('AKS', 'AzureStorage'))"
    default: !expression "Provider == 'GCP' ? 'GKE' : (Provider == 'AWS' ? 'EKS' : 'AKS')"

  - name: K8sClusterName
    description: What is your Kubernetes cluster name
    type: Input
    dependsOnTrue: !expression "Service == 'GKE' || Service == 'EKS' || Service == 'AKS'"

  files:
  - path: xld-k8s-infrastructure.yml
    dependsOnTrue: !expression "Service == 'GKE' || Service == 'EKS' || Service == 'AKS'"
  - path: xld-storage-infrastructure.yml
    dependsOnTrue: !expression "Service == 'CloudStorage' || Service == 'S3' || Service == 'AzureStorage'"
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
| `-b` | `--blueprint` | | `xl blueprint -b aws/monolith`<br>`xl blueprint -b /path/to/local/blueprint/dir`<br/>`xl blueprint -b ../relative/path/to/local/blueprint/dir`  | Looks for the specified absolute or relative folder path in local file system, if not found looks for the path relative to the current remote repository and instead of asking user which blueprint to use, it will directly fetch the specified blueprint from remote repository, or give an error if blueprint not found in both local filesystem and remote repository |
