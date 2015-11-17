import React from 'react'
import ReactDOM from 'react-dom'

import resetPasswordFlow from './stores/reset-password-flow'
import ResetPasswordForm from './ui/reset-password-form'


export default function clientResetPassword() {
  const attachPoint = uidocument.getElementById('form-container')
  const contextData = JSON.parse(attachPoint.getAttribute('data-context'))
  resetPasswordFlow.initData(contextData)

  ReactDOM.render(
    <ResetPasswordForm />,
    attachPoint
  )
}
