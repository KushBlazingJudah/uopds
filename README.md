# uopds

uopds is an OPDS server implementing nothing more than the bare minimum required
to interface with KOreader.

The feature scope is literally nothing more than listing out books in a
directory and making them available to download.

## Supported features

uopds in its current state is *feature complete*; new features will most likely
be rejected, especially if they contain breaking changes that require manual
intervention.
New support for formats not yet supported, and of course bug fixes, are welcome
and appreciated.

- generates an OPDS catalog for files in a directory
- supports EPUB metadata
- most likely doesn't implement the specs correctly

## Running

Build with `go build` and use the `uopds` binary.

uopds by default has everything in its current working directory; it will look
for books in `books` and store its database in `database`.
You can change this by specifying `-books <book folder location>` and `-db
<database location>`.

uopds is meant to have easy support for running behind a reverse proxy that
redirects based on the path.
This is how I use uopds on my server; nginx forwards requests on `/opds` to
uopds.
uopds requires minimal configuration for these setups, as all you need is to
pass `-root <path>`.
If you want to require authentication, you should use a reverse proxy and set it
up through there.
