# xl-up

## Requested resources in K8S production setup

| Deployment            | Memory | CPU | PODs |
|-----------------------|:------:|:---:|:----:|
| Psql                  |   1Gi  | 0.5 |   1  |
| Elasticsearch         |  2.5Gi | 0.5 |   1  |
| Fluentd               |  0.5Gi | 0.1 |  3*  |
| Grafana               |  0.2Gi | 0.1 |   1  |
| Kibana                |   1Gi  | 0.5 |   1  |
| XL Deploy             |   4Gi  |  2  |   2  |
| XL Release            |   4Gi  |  2  |   2  |
| Total: XebiaLabs      |  16Gi  |  8  |   4  |
| Total: K8S-foundation |  6.2Gi | 1.9 |   7  |
| Total: All            | 22.2Gi | 9.9 |  11  |

\* **Fluentd** is a DaemonSet. Thus PODs count depends on your k8s cluster setup. For example: you will have 4 Fluentd pods if you have 4 scheduling nodes in k8s cluster. 
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
