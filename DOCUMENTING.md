Contributing to the Exoscale Terraform Provider documentation
==

Your contributions to the Exoscale Terraform Provider documentation are all welcome.

In fact, thoroughly documenting new features or behaviors is mandatory for all PRs
introducing them.

In order to maintain the documentation uniform, we ask that you follow the following conventions.

Arguments and attributes
--

* All resources or data-sources **names** must be _highlighted_;
  example given: `exoscale_compute_instance`

* When applicable, **resource and ad-hoc data-source** must be _hyperlinked one to another_;
  example given (for the `exoscale_compute_instance` resource):
  _Corresponding data sources: `exoscale_compute_instance`._

* All arguments - parameters passed to resources/data-source - must be documented:

  + **(Required)** arguments are always _first_; their block must be separated from optional ones
    (next) by a _blank line_

  + Then comes optional arguments; **default** value must always be mentioned;
    example given: _(default: `standard.medium`)_

  + When not obvious, mention the argument **type** too, followed by its default if optional;
    example given: _(boolean; default: `true`)_

  + **(Deprecated)** arguments are always _last_; their block must be separated from optional ones
    (above) by a _blank line_

  + Within each block (required, optional, deprecated), arguments must be **ordered** by
    _alphabetical_ order.

  + With the exception of **well-known arguments**, which must always come _first,
    in the following order_:
    - `zone`
    - `id`
    - `name`
    - `description`

* All attributes - values returned by resources/data-sources - must be documented:

  + **(Deprecated)** attributes are always _last_; their block must be separated from regular ones
    by a _blank line_

  + Within each block (regular, deprecated), attributes must be **ordered** by
    _alphabetical_ order.

  + With the exception of **well-known attributes**, which must always come _first,
    in the following order_:
    - `zone`
    - `id`
    - `name`
    - `description`

Usage and use-case examples
--

Resources or data-sources **Usage** examples must be kept as simple as possible, demonstrating the
use of arguments and resources _independently from other resources/data-sources_.

**Use case** examples - mixing different resources and data-sources to achieve a well-identified
purpose - must be stored in the `examples` directory (where they may be hyperlinked from each
individual resource/data-source `docs` pages); example given: _Please refer to the `examples`
directory for complete configuration examples._

Locale and encoding
--

Please use `en_US.UTF-8` locale/encoding, in particular in respect with spell-checking.
