# xl-up

## Run xl up for public repo

`xl up`

## Run xl up from private repo

Until release, xl up blueprint is saved on private repo https://github.com/xebialabs/xl-up-blueprint. If you have access to this repo then you can run the up command after configuring this repo in the `blueprint.repositories` section of the `.xebialabs/config.yaml` file.

```
blueprint:
  repositories:
  - name: xl-up-blueprint
    type: github
    repo-name: xl-up-blueprint
    owner: xebialabs
    branch: beta
    token: YOUR-PERSONAL-GITHUB-TOKEN

``` 

and run :

`xl up --dev`


## Development 

For development purposes you have to specify the branch you are working on on the `~/.xebialabs/conf` :

```
blueprint:
  repositories:
  - name: xl-up-blueprint
    type: github
    repo-name: xl-up-blueprint
    owner: xebialabs
    branch: YOUR-BRANCH
    token: YOUR-PERSONAL-GITHUB-TOKEN

``` 

and run:

`xl up --dev`
