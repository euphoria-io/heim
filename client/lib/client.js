import 'babel-polyfill'
import 'whatwg-fetch'

import clientRoom from './clientRoom'
import clientVerifyEmail from './clientVerifyEmail'
import clientResetPassword from './clientResetPassword'


// setup globals (used by env frame)
window.uiwindow = window.top
window.uidocument = window.top.document

let tag = document.getElementById('heim-js')
if (!tag) {
  // FIXME: fallback to ease update. remove once heim-js id is rolled out for a while.
  const scripts = document.getElementsByTagName('script')
  tag = scripts[scripts.length - 1]
}

const entrypoint = tag.getAttribute('data-entrypoint')
if (!entrypoint) {
  clientRoom()
} else {
  const crashHandler = require('./ui/crashHandler').default
  document.addEventListener('ravenHandle', crashHandler)
  if (entrypoint === 'verify-email') {
    clientVerifyEmail()
  } else if (entrypoint === 'reset-password') {
    clientResetPassword()
  }
}
