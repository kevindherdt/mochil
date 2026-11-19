# Security policy

Mochil serves a private library over a network, so a security report is always
taken seriously — including in a hobby project.

## Reporting

Please **do not open a public issue**. Either:

- use GitHub's private vulnerability reporting on this repository
  (Security → Report a vulnerability), or
- contact the maintainer directly through the address on the GitHub profile.

Include the version, how it is deployed (Compose, reverse proxy in front or not),
what an attacker gains, and the smallest reproduction you have. A proof of
concept is welcome; please do not run it against anyone else's instance.

## What to expect

Acknowledgement as soon as the report is read, an assessment of whether it is a
real issue, and a fix in a normal release. There is no bug bounty and no
guaranteed timeline — this is one person's project.

## Scope

In scope: authentication and session handling, authorization between accounts,
path handling in the library and vault, the OPDS and KOReader endpoints, and
anything that would leak files outside a configured library.

Out of scope: findings that require an already-compromised host, denial of
service by resource exhaustion on your own machine, and reports from automated
scanners with no demonstrated impact.

## Supported versions

Only the latest release. Fixes are not backported; upgrading is one `docker
compose pull`.
