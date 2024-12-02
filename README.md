# nanogit

A limited, cloud-ready Git implementation for use in Grafana.

## Project goals

The following are goals in this project:

* Support reading Git files over HTTPS on github.com.
* Support reading Git trees over HTTPS on github.com.
* Support writing new Git objects over HTTPS on github.com.
* Support writing Git object deltas over HTTPS on github.com.

Eventually, these goals will also be relevant:

* Support any HTTPS Git service that supports `git-upload-pack` on `Git-Protocol: version=2`. Gitlab is a good example of this.

These are non-goals (for now?):

* Supporting the `git://` and Git-over-SSH protocols.
* Supporting the file protocol, i.e. without any wire communications.
* Supporting commit signing.
* Supporting Git signature verification.
* Supporting proper clones.

## Maintenance

This project is not maintained. It was created as part of a hackathon, and as
such is not intended as a real Grafana product, nor in any way or shape or form
intended for production.

## Graduation

If the project is intended to be graduated to a proper OSS project, you should
just replace `hackathon-2024-12-nanogit` with `nanogit` everywhere. Tools like
[sd](https://github.com/chmln/sd) can be of help.

Make sure someone owns and maintains this project if you intend on graduating
it. Otherwise, we may gain security vulnerabilities which we won't solve.

## Resources

Want to learn how Git works? The following resources are useful:

* <https://git-scm.com/book/ms/v2/Git-on-the-Server-The-Protocols>
* <https://git-scm.com/docs/protocol-v2>
* <https://git-scm.com/docs/pack-protocol>
* <https://git-scm.com/docs/git-http-backend>
* <https://git-scm.com/docs/http-protocol>
* <https://git-scm.com/docs/gitprotocol-http>
* <https://git-scm.com/docs/gitprotocol-v2>
* <https://git-scm.com/docs/gitprotocol-pack>
* <https://git-scm.com/docs/gitprotocol-common>