import Raven from 'raven-js'

// Hack: Raven not currently setting window.Raven itself; perhaps this will be
// fixed in a future version. We need to use require() here because imports
// seem to be reordered by babel.
window.Raven = Raven
require('raven-js/plugins/native')
require('raven-js/plugins/console')

Raven.config(process.env.SENTRY_ENDPOINT, {
  release: process.env.HEIM_RELEASE,
  tags: {'git_commit': process.env.HEIM_GIT_COMMIT},
}).install()
