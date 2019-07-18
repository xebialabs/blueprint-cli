## Blueprint schema design v2

```yaml
apiVersion: xl/v2
kind: Blueprint
metadata:
  # not used at the moment, serves as documentation
  name: Test Project 
  # not used at the moment, serves as documentation
  description: Is just a test blueprint project used for manual testing of inputs 
  # not used at the moment, serves as documentation
  author: XebiaLabs 
  # not used at the moment, serves as documentation
  version: 2.0
  suppressXebiaLabsFolder: false
  instructions: This is instructions

# rename !expression to !expr in v2
# merge !fn functions into expression functions and remove !fn support in v2
# get rid of the backward compatibility code for spec -> parameters in v2
spec:
  parameters:
  # A parameter with question
  - name: AppName # mandatory field (Validated)
    prompt: What is your application name? # mandatory when there is no value field (Validated)
    description: this is your fancy description # will be shown as question hint if set
    label: Application Name # defaults to name, not mandatory
    type: Input # mandatory when there is no value field (Validated)
    default: Foo # should not be set when value is set (Validated)
    options: # supports both text and label/value. label/value will be merged as label-value (Validated)
    - eu-north
    - label: EU West # no !expr support
      value: eu-west1 # no !expr support
    validate: !expr "regex('[a-z]', AppName) && AppName != 'admin'"
    saveInXlvals: true # case changed for v2
    promptIf: # renamed from dependsOn, use expressions for dependsOnFalse use case (promptIf: !expr "!Foo" ) (Validated)
  # A secret parameter with question 
  - name: Password
    prompt: What is your Password?
    description: this is your fancy description
    label: Password
    type: SecretInput # a new type along with SecretEditor & SecretFile
    replaceAsIs: false # renamed from useRawValue, can be used only with SecretInput, SecretEditor & SecretFile (Validated)
    revealOnSummary: false # renamed from showValueOnSummary, can be used only with SecretInput, SecretEditor & SecretFile (Validated)
    default: Foo
  # A parameter with value 
  - name: TestDepends # mandatory field (Validated)
    # If value is specified, you can't have prompt, promptIf, default & options (Validated)
    value: FOO

  files:
  - path: xld-environment.yml.tmpl
  - path: xld-infrastructure.yml.tmpl
  - path: xlr-pipeline-2.yml
    writeIf: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends" # renamed from dependsOn
  - path: xlr-pipeline.yml
    writeIf: TestDepends # renamed from  dependsOn
  - path: xlr-pipeline-3.yml
    renameTo: xlr-pipeline-4.yml # can be used to give a different name to output file

# New compose block for v2
# includeBefore/includeAfter will decide if the blueprint should be composed before or after the master blueprint, this will affect the order of question and order in which files are written, multiple before/after will stack based on order of definition.
  includeBefore:
  # 'blueprint' we will look in the current-repository being used for these
  - blueprint: kubernetes/gke-cluster 
    # To use parameters in includeIf they need to be defined before the expression is evaluated.
    includeIf: # include on condition
    # 'parameterOverrides' we can provide values for any parameter in the blueprint being composed. This way we can force to skip any question by providing a value for it
    parameterOverrides: # can override everything except name and type fields
    - name: Foo
      value: hello 
    - name: bar
      value: true

    # 'fileOverrides' can be used to skip or rename files and can be conditional using dependsOn
    fileOverrides: # can override everything except path field
    - path: xld-infrastructure.yml.tmpl
      writeIf: !expr "false" # skip files by using an expression that evaluates to false
    - path: xld-infrastructure.yml.tmpl
      renameTo: xld-infrastructure2.yml

 includeAfter:
  - blueprint: kubernetes/namespace
    # To use parameters in includeIf they need to be defined before the expression is evaluated.
    includeIf: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
    parameterOverrides:
    - name: Foo
      value: hello
```
