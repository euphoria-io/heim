import 'babel-polyfill'
import 'whatwg-fetch'

import clientRoom from './client-room'
import clientVerifyEmail from './client-verify-email'
import clientResetPassword from './client-reset-password'


// setup globals (used by env frame)
window.uiwindow = window.top
window.uidocument = window.top.document

const tag = document.getElementById('heim-js')
const entrypoint = tag.getAttribute('data-entrypoint')
if (!entrypoint) {
  clientRoom()
} else {
  const crashHandler = require('./ui/crash-handler').default
  document.addEventListener('ravenHandle', crashHandler)
  if (entrypoint === 'verify-email') {
    clientVerifyEmail()
  } else if (entrypoint === 'reset-password') {
    clientResetPassword()
  }
}
