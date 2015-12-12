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

const origCaptureException = Raven.captureException
window.Raven.captureException = function captureException(ex, options) {
  const newOptions = options || {}
  if (ex.action) {
    newOptions.tags = newOptions.tags || {}
    newOptions.tags.action = ex.action
  }
  if (ex.response) {
    newOptions.extra = newOptions.extra || {}
    newOptions.extra.response = ex.response
  }
  origCaptureException.call(Raven, ex, newOptions)
}
