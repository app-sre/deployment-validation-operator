# Conventions for OSD operators written in Go

This convention is suitable for both cluster- and hive-deployed operators.

The following components are included:

## `make` targets and functions.
**Note:** Your repository's main `Makefile` needs to be edited to include the
"nexus makefile include":

```
include boilerplate/generated-includes.mk
```

One of the primary purposes of these `make` targets is to allow you to
standardize your prow and app-sre pipeline configurations. They should be as
follows:

### Prow

| Test name / `make` target | Purpose                                                                                                         |
|---------------------------|-----------------------------------------------------------------------------------------------------------------|
| `validate`                | Ensure code generation has not been forgotten; and ensure generated and boilerplate code has not been modified. |
| `lint`                    | Perform static analysis.                                                                                        |
| `test`                    | "Local" unit and functional testing.                                                                            |
| `coverage`                | (Code coverage)[#code-coverage] analysis and reporting.                                                         |
| `build`                   | Code compilation and bundle generation.                                                                         |

In addition to configuring these test targets, make sure your
`build_root` stanza is configured to use the configuration from your
repository, which is provided by this convention:

```yaml
build_root:
  from_repository: true
```

### app-sre

The `build-push` target builds and pushes the operator and OLM registry images,
ready to be SaaS-deployed.

## Code coverage
- A `codecov.sh` script, referenced by the `coverage` `make` target, to
run code coverage analysis per [this SOP](https://github.com/openshift/ops-sop/blob/93d100347746ce04ad552591136818f82043c648/services/codecov.md).

- A `.codecov.yml` configuration file for
  [codecov.io](https://docs.codecov.io/docs/codecov-yaml). Note that
  this is copied into the repository root, because that's
  [where codecov.io expects it](https://docs.codecov.io/docs/codecov-yaml#can-i-name-the-file-codecovyml).

## Linting and other static analysis with `golangci-lint`

- A `go-check` `make` target, which
- ensures the proper version of `golangci-lint` is installed, and
- runs it against
- a `golangci.yml` config.

## Checks on generated code

The convention embeds default checks to ensure generated code generation is current, committed, and unaltered.
To trigger the check, you can use `make generate-check` provided your Makefile properly includes the boilerplate-generated include `boilerplate/generated-includes.mk`.

Checks consist of : 
* Checking all files are committed to ensure a safe point to revert to in case of error
* Running the `make generate` command to regenerate the needed code
* Checking if this results in any new uncommitted files in the git project or if all is clean.

## More coming soon
