## 0.3.0 (Not released)

- Added `tau fmt` command to format tau files
- Fixed execution order. Could sometimes execute dependencies too late.

## 0.2.0 (22. September 2019)

Improved merging of blocks. Previously map attributes in inputs block with same name would cause an error with duplicate attributes. With this release it will merge the maps together.

`common_auto.hcl`

```terraform
inputs {
    tags = {
        costCenter = "IT"
        resource = "Kubernetes"
    }
}
```

`kubernetes.hcl`

```terraform
inputs {
    tags = {
        responsible = "noreply@email.com"
    }
}
```

Merging these 2 files together will now result in a map with `costCenter`, `resource` and `responsible` all defined.

FEATURES:

* Support merging items in input maps together [#13](https://github.com/avinor/tau/issues/13)

IMPROVEMENTS:

* Checks that `environment_variables` are not maps or lists
* Improved merging of `backend`, `environment_variables` and `inputs`.

## 0.1.0 (18. September 2019)

First release that can be used for deployments in pipeline. This is still a bit work in progress, but stable enough to include in deployments scripts.