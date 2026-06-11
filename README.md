# mongo-go-driver-exp

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> **⚠ Experimental — do not use in production**
>
> This repository contains experimental Go modules that extend the [MongoDB Go
> Driver](https://github.com/mongodb/mongo-go-driver). All APIs are **unstable**
> and may change, be renamed, or be removed entirely without prior notice and
> without a major version bump. These modules are **not covered** by the MongoDB
> Go Driver's compatibilty guarantees. Use them for evaluation and feedback
> only.

## About

`mongo-go-driver-exp` is a staging ground for features, APIs, and add-ons that
are not yet ready for the main [MongoDB Go
Driver](https://github.com/mongodb/mongo-go-driver) module. Ideas start here,
get real-world feedback, and are either promoted to the main driver or retired.
If you try something from this repo, [filing an
issue](https://github.com/mongodb-labs/mongo-go-driver-exp/issues) with your
experience helps shape what graduates.

## Modules

| Module | Description | Status |
|--------|-------------|--------|
| *(none yet)* | | |

More modules are on the way. Watch this repo or check back soon.

## Installation

Each directory in this repository is its own Go module. Once modules are
available, install them individually:

```bash
go get github.com/mongodb-labs/mongo-go-driver-exp/<module-name>@latest
```

## Contributing & Feedback

Issues and pull requests are welcome. For code style and conventions,
follow the guidelines in the MongoDB Go Driver's
[contributing guide](https://github.com/mongodb/mongo-go-driver/blob/master/CONTRIBUTING.md).

Keep in mind:

- Experimental modules may be **promoted** to the main driver, **kept here**
  long-term as optional add-ons, or **deprecated** with no replacement.
- There is no SLA on issues in this repository.
- Breaking changes can happen at any time.

## License

Apache 2.0 — see [LICENSE](LICENSE).
