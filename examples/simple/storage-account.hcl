module {
    source = "avinor/storage-account/azurerm"
    version = "1.1.0"
}

inputs {
    name = "simple"
    resource_group_name = "simple-rg"
    location = "westeurope"

    containers = [
        {
            name = "test"
            access_type = "private"
        },
    ]
}