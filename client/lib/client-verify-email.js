import React from 'react'
import ReactDOM from 'react-dom'

import verifyEmailFlow from './stores/verify-email-flow'
import VerifyEmailForm from './ui/verify-email-form'


export default function clientVerifyEmail() {
  const attachPoint = uidocument.getElementById('form-container')
  const contextData = JSON.parse(attachPoint.getAttribute('data-context'))
  verifyEmailFlow.initData(contextData)

  ReactDOM.render(
    <VerifyEmailForm />,
    attachPoint
  )
}
