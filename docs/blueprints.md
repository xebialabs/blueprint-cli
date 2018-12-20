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
| **type** | `Input`/`Select`/`Confirm` | | - | y | Type of the prompt input |
| **value** | - | eu-west-1<br>!fn aws.regions(ecs)[0] | - | n | If present, user will not be asked a question to provide value |
| **default** | - | Awesome App | - | n | Default value, will be present during the question prompt. Also will be the variable value if question is skipped. |
| **description** | - | Application name, will be used in various AWS resource names | - | n | If present, will be used instead of default question text |
| **secret** | `true`/`false` | - | `false` | n | Variables that are marked as secret are saved in `secrets.xlvals` files so that they won't be checked in GIT repo and will not be replaced by default in the template files |
| **options** | - | - eu-west-1<br>- us-east-1<br>- us-west-1 | - | n<br>(required for `Select` input type) | Set of options for the `Select` input type. Can consist of any number of text values or custom tags |
| **pattern** | - | [a-z-]* | - | n | Validation regular expression, to be verified at the time of user input |
| **dependsOnTrue** | - | CreateNewCluster<br>!fn aws.credentials().IsAvailable | - | n | If this question is need to be asked to user depending on the value of another, dependsOn field can be defined.<br>A valid variable name should be given and the variable name used should have been defined before order-wise. Function tags also can be used, but expected result should always be boolean. |
| **dependsOnFalse** | - | CreateNewCluster<br>!fn aws.credentials().IsAvailable | - | n | Reverse logic for dependsOn, see above
| **saveInXlVals** | `true`/`false` | - | `true` for secret fields<br>`false` for other fields | n | If true, output variable will be included in the `values.xlvals` output file. By default every secret field will be written to `secrets.xlvals` file and this setting doesn't effect that functionality |

#### Files Fields

| Field Name | Expected value(s) | Examples | Default Value | Required | Explanation |
|:--------------: |:--------------------: |------------------------------------------------------------ |:-------------: |:---------------------------------------: |------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **path** | - | xebialabs/xlr-pipeline.yaml | - | y | File/template path to be copied/processed  |
| **dependsOnTrue** | - | CreateNewCluster<br>!fn aws.credentials().IsAvailable | - | n | This file will be generated only when value of a variable or function return true.<br>A valid variable name should be given and the variable name used should have been defined. Function tags also can be used, but expected result should always be boolean. |
| **dependsOnFalse** | - | CreateNewCluster<br>!fn aws.credentials().IsAvailable | - | n | This file will be generated only when value of a variable or function return false.<br>A valid variable name should be given and the variable name used should have been defined. Function tags also can be used, but expected result should always be boolean.|

## Supported YAML Tags

### Custom function tag [!fn]

Blueprints support custom functions to be used within variable definitions (`spec` part in YAML file).

Custom function syntax: `DOMAIN . MODULE (PARAMETERS...) . ATTRIBUTE|[INDEX]`

Available custom functions:

| Domain | Module | Examples | Parameters | Attributes/Index | Description |
|:------: |:-----------: |:----------------------------------------: |:----------------: |------------------------------------------------------- |:------------------------------------------------------------: |
| **aws** | **credentials** | aws.credentials().AccessKeyID | [optional] Profile name | AccessKeyID<br>SecretAccessKey<br>SessionToken<br>ProviderName<br>IsAvailable | Read AWS credentials package from system aws-cli config file |
| **aws** | **regions** | aws.regions(ecs)<br>aws.regions(ecs)[0] | AWS service ID | Any index of the resulting array | Get list of available regions for the specified AWS service |


## Go Templates

In blueprint template files using `.tmpl` extension, GoLang templating can be used. 
Please refer to the following [cheatsheet](https://curtisvermeeren.github.io/2017/09/14/Golang-Templates-Cheatsheet) for more details how to use GoLang templates. 
Also support for additional [Sprig](http://masterminds.github.io/sprig/) functions are included in the templating engine, as well as list of custom XL functions. 
Please refer to below table for additional functions available.

| Function | Example | Description |
|:---------: |:----------------------: |:-------------------------------------------------: |
| kebabcase | `.AppName | kebabcase` | Convert string to use kebab case (separated by -) |