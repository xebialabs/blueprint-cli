## Blueprint schema design v1

```yaml
apiVersion: xl/v1
kind: Blueprint
metadata:
  # not used at the moment, serves as documentation
  projectName: Test Project 
  # not used at the moment, serves as documentation
  description: Is just a test blueprint project used for manual testing of inputs 
  # not used at the moment, serves as documentation
  author: XebiaLabs 
  # not used at the moment, serves as documentation
  version: 1.0 
  instructions: This is instructions

parameters: see spec > parameters
files: see spec > files

spec:
  parameters:
  # A parameter with question
  - name: AppName # mandatory field (Validated)
    description: this is your fancy description # will be shown as question if set else name will be used for question
    type: Input # mandatory (Validated)
    default: Foo
    options: # supports both text and expression/fn. Mandatory for Select type (Validated)
    - eu-north
    - !fn aws.regions()
    saveInXlVals: true
    pattern: "[a-z]*"
    dependsOnTrue: TestDepends # use expressions, fn or variable name
    dependsOnFalse: TestDepends # use expressions, fn or variable name
  # A secret parameter with question 
  - name: Password
    description: What is your Password?
    type: Input
    secret: true
    useRawValue: false
    showValueOnSummary: false
    default: Foo
  # A parameter with value 
  - name: TestDepends # mandatory field (Validated)
    type: Input
    value: FOO

  files:
  - path: xld-environment.yml.tmpl
  - path: xld-infrastructure.yml.tmpl
  - path: xlr-pipeline-2.yml
    dependsOnTrue: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
    dependsOnFalse: !expression "ExpTest1 == 'us-west' && AppName != 'foo' && TestDepends"
  - path: xlr-pipeline.yml
  - path: xlr-pipeline-3.yml
```
