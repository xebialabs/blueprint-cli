## Blueprint schema design

```yaml
apiVersion: xl/v1
kind: Blueprint
metadata:
  name: Test Project
  description: Is just a test blueprint project used for manual testing of inputs
  author: XebiaLabs
  version: 1.0
  instructions: This is instructions

# merge !fn functions into expressions
# rename !expression to !expr
# get rid of the backward compatibility code for spec->parameters
spec:
  parameters:
  - name: AppName # mandatory
    prompt: What is your application name? # mandatory when there is no value
    description: this is your fancy description # will be shown as question hint if available
    label: Application Name # defaults to name, not mandatory
    type: Input # mandatory when there is no value
    default: Foo # can't be used with value
    options:
    - eu-north
    - label: EU West
      value: eu-west1
    # pattern: move to validate
    validate: !expr "regex('[a-z]', AppName) && AppName != 'admin'"
    saveInXlVals: true
    # remove dependsOnTrue and dependsOnFalse
    dependsOn: 
  - name: Password # mandatory
    prompt: What is your Password? # mandatory when value is not provided
    description: this is your fancy description
    label: Password # defaults to name, not mandatory
    type: 
        name: SecretInput # mandatory when there is no value
        useRawValue: false
        revealOnSummary: false
    default: Foo # can't be used with value
    secret:
    pattern:
    validate:
    dependsOn:
  - name: AppName
    type: Input
    default: foo
  - name: TestDepends
    type: Confirm
  - name: TestDepends
    # add validation(If value is specified, you cant have type, defaults & dependsOn)
    value: FOO

  files:
  - path: xld-environment.yml.tmpl
  - path: xld-infrastructure.yml.tmpl
  - path: xlr-pipeline-2.yml
    dependsOnTrue: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
  - path: xlr-pipeline.yml
    dependsOnFalse: TestDepends
  - path: xlr-pipeline-3.yml

# New compose block
  include: 
  # 'blueprint' we will look in the current-repository being used for these
  - blueprint: kubernetes/gke-cluster 
    # 'stage' will decide if the blueprint should be composed before or after the master blueprint, this will affect the order of question and order in which files are written, multiple before/after will stack based on order of definition.
    stage: before 
    # 'parameterOverrides' we can provide values for any parameter in the blueprint being composed. This way we can force to skip any question by providing a value for it, these can be conditional using dependsOn
    parameterOverrides: 
    - name: Foo
      # expression and functions will be supported for 'value'
      value: hello 
    #   dependsOn: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends" # Not decided yet
    - name: bar
      value: true

    # 'fileOverrides' can be used to skip or rename files and can be conditional using dependsOn
    fileOverrides:
    - path: xld-infrastructure.yml.tmpl
      operation: skip
    #   dependsOn: TestDepends # Not decided yet
    - path: xld-infrastructure.yml.tmpl
      operation: rename
      renamedPath: xld-infrastructure2.yml
    #   dependsOn: TestDepends # Not decided yet

  - blueprint: kubernetes/namespace
    # To use parameters in dependsOn they need to be defined before the expression is evaluated.
    dependsOn: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
    stage: after
    parameterValues:
    - name: Foo
      value: hello
```

Phase 1: Implement the after stage, no support for dependsOn now
Phase 2: Implement the before stage
Phase 3: Implement support for dependsOn 
