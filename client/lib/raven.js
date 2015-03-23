Raven = require('raven-js')
require('raven-js/plugins/native')
require('raven-js/plugins/console')
Raven.config(process.env.SENTRY_ENDPOINT, {
  release: process.env.HEIM_RELEASE,
  tags: {'git_commit': process.env.HEIM_GIT_COMMIT},
}).install()
