# Terraform Packer Provider


A provider for HashiCorp Packer that has Packer embedded in it so that you can run it
on any environment (including Terraform Cloud).

## Documentation

You can find documentation in the [Terraform Registry](https://registry.terraform.io/providers/toowoxx/packer/latest/docs).

The main resource of this provider is [packer_image](https://registry.terraform.io/providers/toowoxx/packer/latest/docs/resources/image) which builds the image using packer.

## Examples

Examples can be found in the [examples subdirectory](examples/).

## Gotchas

### Image management

Packer does not manage your images – which means that neither does this provider.
This provider will **not** detect whether the image exists on the remote because that's
not something that Packer can do.

Terraform providers are only a means of plugging an API or an external system into Terraform
which is what this provider is doing.
Regardless, we still reserve the possibility that we may add support for managing images independently
of Packer itself.

You have multiple options for managing your images:

 * Use data sources and, if necessary, the manifest post-processor in Packer
 * Import state of the created image after successful deployment
 * Manually manage images, for example, by deleting them from your cloud provider or system (for example, you can delete images manually from Azure using the Azure Portal)

You can use the `force` attribute of resource `packer_image` to overwrite the image every time.

The remote state does not affect this provider's ability to function. If you delete an image remotely, Packer will still run and attempt to create a new one which should succeed. There is no fundamental difference between "Creation" and "Update" of a `packer_image` resource.

## Custom Packer Binary

If you prefer to use an external Packer binary instead of the embedded one, set the provider attribute `packer_binary` to the absolute path of your Packer executable:

```
provider "packer" {
  packer_binary = "/usr/local/bin/packer"
}
```

The provider validates the binary by running `packer version`. When unset, the embedded Packer is used.
Due to licensing constraints (BUSL), we are not going to be updating the embedded Packer version past the pre-BUSL commit around the 1.10.0 release.

## License

[Mozilla Public License v2.0](LICENSE)
