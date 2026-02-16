# Gate Scripts

This folder contains CLI scripts that control the gating process for production releases.

## What is "The Gate"?

"Running the gate" refers to executing a series of automated, procedural checks and that ensure the
repository is ready for release. The gate are all the steps of validation and preparation for
release of the repository. It is an essential part of the SDLC process.

## Components

- This `gate` folder defines executable gate scripts with documentation in the headers
- [`shflags`](./shflags/CLAUDE.md) - use shflags for arguments parsing: `. ./shflags/shflags.sh`

---

## Development Guidelines

Please read [CLAUDE.md](./CLAUDE.md).
