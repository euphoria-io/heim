# ◓

Heim is the backend and frontend of [euphoria](https://euphoria.io). The
backend is a Go server that speaks JSON over WebSockets, persisting data to
PostgreSQL. Our web client is built in React/Reflux.

**Currently, heim is released in a pre-alpha state**. Please be advised that
new development is currently being prioritized over stability. We're releasing
in this form because we want to open up our codebase and development progress.
We will make breaking changes to the protocol, and will be slow to merge
complex pull requests while we get our core building blocks in place.


## Getting started

1. Install `git`, [`docker`](https://docs.docker.com/installation/), and
   [`docker-compose`](https://docs.docker.com/compose/install/).

2. Link in our dependencies repository:
    1. Clone [heim-deps](https://github.com/euphoria-io/heim-deps).
    2. `./heim-deps/deps.sh link ./path/to/heim/repo`

3. Build the client static files: `docker-compose run frontend`.

4. Init your db: `docker-compose run upgradedb sql-migrate up`.

5. Start the server: `docker-compose up backend`.

Heim is now running on port 8080. \o/

## Discussion

Questions? Feedback? Ideas? Come join us in
[&euphoria](https://euphoria.io/room/euphoria) or email hi@euphoria.io.

## Licensing

Most of the server is distributed under the terms of the GNU Affero General Public License 3.0.

The client is distributed under the terms of the MIT license.

Art and documentation are distributed under the terms of the CC-BY 4.0 license.

See [LICENSE.md](LICENSE.md) for licensing details.
