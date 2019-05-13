## Blueprint schema design

```yaml
apiVersion: xl/v2
kind: Blueprint
metadata:
  name: Test Project # not used at the moment, serves as documentation
  # not used at the moment, serves as documentation
  description: Is just a test blueprint project used for manual testing of inputs 
  author: XebiaLabs # not used at the moment, serves as documentation
  version: 2.0 # not used at the moment, serves as documentation
  instructions: This is instructions

# rename !expression to !expr
# merge !fn functions into expression functions and remove !fn support
# get rid of the backward compatibility code for spec -> parameters
spec:
  parameters:
  # A parameter with question
  - name: AppName # mandatory field (Need validation)
    prompt: What is your application name? # mandatory when there is no value field (Need validation)
    description: this is your fancy description # will be shown as question hint if set
    label: Application Name # defaults to name, not mandatory
    type: Input # mandatory when there is no value field (Need validation)
    default: Foo # should not be set when value is set (Need validation)
    options: # supports both text and label/value. label/value will be merged as label-value
    - eu-north
    - label: EU West
      value: eu-west1
    pattern: # remove and move to validate
    validate: !expr "regex('[a-z]', AppName) && AppName != 'admin'"
    saveInXlvals: true # case changed
    dependsOnTrue: # remove
    dependsOnFalse: # remove
    promptIf: # (dependsOn) use expressions for dependsOnFalse (promptIf: !expr "!Foo" )
  # A secret parameter with question 
  - name: Password
    prompt: What is your Password?
    description: this is your fancy description
    label: Password
    type: SecretInput # a new type
    replaceAsIs: false # revisit if we find a better name
    revealOnSummary: false
    default: Foo
    # secret:
  # A parameter with value 
  - name: TestDepends # mandatory field (Need validation)
    # If value is specified, you cant have type, prompt, promptIf, default, options & dependsOn (Need validation)
    value: FOO
    promptIf: # not allowed for value fields

  files:
  - path: xld-environment.yml.tmpl
  - path: xld-infrastructure.yml.tmpl
  - path: xlr-pipeline-2.yml
    writeIf: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends" # (dependsOn)
  - path: xlr-pipeline.yml
    writeIf: TestDepends # (dependsOn)
  - path: xlr-pipeline-3.yml

# New compose block
  includeBefore: #changed
  # 'blueprint' we will look in the current-repository being used for these
  - blueprint: kubernetes/gke-cluster 
    # 'stage' will decide if the blueprint should be composed before or after the master blueprint, this will affect the order of question and order in which files are written, multiple before/after will stack based on order of definition.
    # stage: before 
    # 'parameterOverrides' we can provide values for any parameter in the blueprint being composed. This way we can force to skip any question by providing a value for it, these can be conditional using dependsOn
    includeIf: # include on condition
    parameterOverrides: # can override everything except name and type
    - name: Foo
      # expression and functions will be supported for 'value'
      value: hello 
    - name: bar
      value: true

    # 'fileOverrides' can be used to skip or rename files and can be conditional using dependsOn
    fileOverrides:
    - path: xld-infrastructure.yml.tmpl
      writeIf: false
    - path: xld-infrastructure.yml.tmpl
      renameTo: xld-infrastructure2.yml

includeAfter: #changed
  - blueprint: kubernetes/namespace
    # To use parameters in dependsOn they need to be defined before the expression is evaluated.
    includeIf: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
    # stage: after
    parameterOverrides:
    - name: Foo
      value: hello
```

Phase 1: Implement the after stage, no support for dependsOn now
Phase 2: Implement the before stage
