# Terraform Provider for Packer


A Terraform provider for use with Packer. It has Packer embedded in it so that you can run it
on any environment (including Terraform Cloud).

This provider is an independent open source project. It is not affiliated with,
sponsored by, or endorsed by HashiCorp. See the [Trademark Notice](#trademark-notice) below.

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

Alternatively, set `packer_binary_url` to download a Packer-compatible binary from a URL of your choice.
The URL may serve a raw executable or a zip archive containing one. Downloads are cached locally and
reused; changing the URL or checksum triggers a fresh download. Use `packer_binary_checksum` to verify
the downloaded artifact:

```
provider "packer" {
  packer_binary_url      = "https://example.com/dist/packer_1.9.2_linux_amd64.zip"
  packer_binary_checksum = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}
```

`packer_binary` and `packer_binary_url` are mutually exclusive. The provider validates the binary by
running `packer version`. When both are unset, the embedded Packer is used.

This provider is an independent project and is not affiliated with, sponsored by, or endorsed
by HashiCorp. When you point `packer_binary_url` at a download, you are responsible for choosing
a trustworthy source and for complying with the license of the binary it serves. The provider
neither distributes nor recommends any particular binary.

Due to licensing constraints (BUSL), we are not going to be updating the embedded Packer version past the pre-BUSL commit around the 1.10.0 release.

## Embedded Packer

This provider embeds a build of Packer that is Copyright (c) HashiCorp, Inc. and
licensed under the Mozilla Public License 2.0. The embedded build is based on upstream
commit [`4d0a51c`](https://github.com/hashicorp/packer/commit/4d0a51c1892ea91f5eb2d5f56fabe66d729b31d2)
(August 2023), a development commit that predates both the relicensing to BUSL and the
1.10.0 release (it reports version `1.10.0-mpl`). The complete source code of the
embedded build is available at [github.com/toowoxx/packer](https://github.com/toowoxx/packer),
tag `v1.10.0-toowoxx.custom.104`.

When run in embedded Packer mode, the binary prints this attribution to stderr.

## Trademark Notice

HashiCorp, Packer, and Terraform are trademarks or registered trademarks of HashiCorp, Inc.
and/or its affiliates. All other trademarks are the property of their respective owners.

These names are used in this project solely to identify the software the provider
interoperates with (nominative fair use) and to follow the naming convention that the
Terraform Registry requires for providers (`terraform-provider-<name>`). Their use does not
imply any affiliation with, sponsorship by, or endorsement by HashiCorp. This project is
developed and maintained independently, and no trademark rights are claimed in any of
these names.

## License

[Mozilla Public License v2.0](LICENSE)
