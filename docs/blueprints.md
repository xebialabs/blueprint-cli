# Blueprints

## Root YAML Fields

| Field Name | Expected value | Examples | Required |
|:----------: |:--------------: |:---------: |:--------: |
| **apiVersion** | xl/v1 | - | y |
| **kind** | Blueprint | - | y |
| **metadata** | - | see below | n |
| **spec** | - | see below | y |

### Metadata Fields

| Field Name | Expected value | Examples | Required |
|:-----------: |:--------------: |--------------------------------------------------------- |:--------: |
| **projectName** | - | Sample Project | n |
| **description** | - | A long description for describing the blueprint project | n |
| **author** | - | XebiaLabs | n |
| **version** | - | 1.0 | n |

### Spec fields

The spec field holds parameters and files

#### Parameters Fields

| Field Name | Expected value(s) | Examples | Default Value | Required | Explanation |
|:--------------: |:--------------------: |------------------------------------------------------------ |:-------------: |:---------------------------------------: |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **name** | - | AppName | - | y | Variable name, to be used in templates |
| **type** | `Input`/`Select`/`Confirm`/`Editor`/`File` | | - | y | Type of the prompt input |
| **value** | - | - eu-west-1<br>-`!fn aws.regions(ecs)[0]` | - | n | If present, user will not be asked a question to provide value. "yes" or "no" with quotes should be used in Confirm type variables. |
| **default** | - | Awesome App | - | n | Default value, will be present during the question prompt. Also will be the variable value if question is skipped. |
| **description** | - | Application name, will be used in various AWS resource names | - | n | If present, will be used instead of default question text |
| **secret** | `true`/`false` | - | `false` | n | Variables that are marked as secret are saved in `secrets.xlvals` files so that they won't be checked in GIT repo and will not be replaced by default in the template files |
| **options** | - | - eu-west-1<br>- us-east-1<br>- us-west-1 | - | n<br>(required for `Select` input type) | Set of options for the `Select` input type. Can consist of any number of text values or custom tags |
| **pattern** | - | `[a-z-]* `| - | n | Validation regular expression, to be verified at the time of user input |
| **dependsOnTrue** | - | - CreateNewCluster<br>- `!fn aws.credentials().IsAvailable`<br>- `!expression "CreateNewCluster == true"` | - | n | If this question is need to be asked to user depending on the value of another, dependsOn field can be defined.<br>A valid variable name should be given and the variable name used should have been defined before order-wise. Function tags also can be used, but expected result should always be boolean. |
| **dependsOnFalse** | - | - CreateNewCluster<br>- `!fn aws.credentials().IsAvailable`<br>- `!expression "CreateNewCluster == true"` | - | n | Reverse logic for dependsOnTrue, see above
| **saveInXlVals** | `true`/`false` | - | `true` for secret fields<br>`false` for other fields | n | If true, output variable will be included in the `values.xlvals` output file. By default every secret field will be written to `secrets.xlvals` file and this setting doesn't effect that functionality |

> Note: `File` type doesn't support `value` parameter. `default` parameter for this field expects to have a file path instead of final value string.

#### Files Fields

| Field Name | Expected value(s) | Examples | Default Value | Required | Explanation |
|:--------------: |:--------------------: |------------------------------------------------------------ |:-------------: |:---------------------------------------: |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **path** | - | `xebialabs/xlr-pipeline.yaml` | - | y | File/template path to be copied/processed  |
| **dependsOnTrue** | - | - CreateNewCluster<br>- `!fn aws.credentials().IsAvailable`<br>- `!expression "CreateNewCluster == true"` | - | n | This file will be generated only when value of a variable or function return true.<br>A valid variable name should be given and the variable name used should have been defined. Function tags also can be used, but expected result should always be boolean. |
| **dependsOnFalse** | - | - CreateNewCluster<br>- `!fn aws.credentials().IsAvailable`<br>- `!expression "CreateNewCluster == true"` | - | n | This file will be generated only when value of a variable or function return false.<br>A valid variable name should be given and the variable name used should have been defined. Function tags also can be used, but expected result should always be boolean.|

## Supported Custom YAML Tags

### Function tag (`!fn`)

Blueprints support custom functions to be used within variable definitions and file declarations (`spec` part in YAML file). Function tag can be used in `value`, `default`, `options`, `dependsOnTrue` and `dependsOnFalse` fields.

Custom function syntax: `!fn DOMAIN.MODULE (PARAMETERS...).ATTRIBUTE|[INDEX]`

#### Available custom functions:

| Domain | Module | Examples | Parameters | Attributes/Index | Description |
|:------: |:-----------: |:----------------------------------------: |:----------------: |------------------------------------------------------- |:------------------------------------------------------------: |
| **aws** | **credentials** | `aws.credentials().AccessKeyID` | [optional] Profile name | - AccessKeyID<br>- SecretAccessKey<br>- SessionToken<br>- ProviderName<br>- IsAvailable | Read AWS credentials package from system aws-cli config file |
| **aws** | **regions** | `aws.regions(ecs)`<br>`aws.regions(ecs)[0]` | AWS service ID | Any index of the resulting array | Get list of available regions for the specified AWS service |

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

Only supported types are; `float64`, `bool`, `string`, and `arrays`.

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



## Go Templates

In blueprint template files using `.tmpl` extension, GoLang templating can be used. 
Please refer to the following [cheatsheet](https://curtisvermeeren.github.io/2017/09/14/Golang-Templates-Cheatsheet) for more details how to use GoLang templates. 
Also support for additional [Sprig](http://masterminds.github.io/sprig/) functions are included in the templating engine, as well as list of custom XL functions. 
Please refer to below table for additional functions available.

| Function | Example | Description |
|:---------: |:----------------------: |:-------------------------------------------------: |
| kebabcase | `.AppName | kebabcase` | Convert string to use kebab case (separated by -) |


## Blueprint Repository

[TODO]

## Blueprint CLI Flags

[TODO]
