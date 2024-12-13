# Flight School

A web app for tracking confidence in knowledge of the
[Airman Certification Standards (ACS)][acs]. At a high level, each ACS is
broken down into areas which contain tasks which contain elements. This app
helps track which parts of the ACS you have the least confidence in so you can
study efficiently.

## Prerequisites

The application expects to have a Postgres database available.

## Build It

Build tasks are orchestrated with [`just`][just]. The main entrypoint for
building the application is:
```shell
just build
```

Other tasks can be viewed via `just --list`.

## Run It

The main web application can be built from `./cmd/flight-school`. It launches a
web server that listens to `0.0.0.0:8000`.

```text
Run the flight-school web server

Usage:
  flight-school [flags]

Flags:
      --debug        Enable debug behavior
      --dsn string   DSN for connecting to the database ($FLIGHT_SCHOOL_DSN)
  -h, --help         help for flight-school
```

There's an additional program in `./cmd/populate-acs` that is useful for
populating the database with the contents of a particular ACS. It accepts the
path to a JSON file as a single positional argument.

```text
Usage of populate-acs:
  -dsn string
        DSN for the database (default "postgres://localhost")
```

[acs]: https://www.faa.gov/training_testing/testing/acs
[just]: https://github.com/casey/just
