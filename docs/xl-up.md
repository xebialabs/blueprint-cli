# xl-up

## Run xl up for public repo

`xl up`

## Run xl up from private repo

Until release xl up blueprint is saved on https://github.com/xebialabs/xl-up-blueprint. If you have access to this repo then you can run the up command after configuring this repo in the `blueprint.repositories` section of the `.xebialabs/config.yaml` file.

```
blueprint:
  repositories:
  - name: xl-up-blueprint
    type: github
    repo-name: xl-up-blueprint
    owner: xebialabs
    branch: LOVE-665
    token: YOUR-PERSONAL-GITHUB-TOKEN

``` 

`xl up --blueprint-current-repository xl-up-blueprint`
