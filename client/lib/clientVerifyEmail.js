import React from 'react'
import ReactDOM from 'react-dom'

import verifyEmailFlow from './stores/verifyEmailFlow'
import VerifyEmailForm from './ui/VerifyEmailForm'


export default function clientVerifyEmail() {
  const attachPoint = uidocument.getElementById('form-container')
  const contextData = JSON.parse(attachPoint.getAttribute('data-context'))
  verifyEmailFlow.initData(contextData)

  ReactDOM.render(
    <VerifyEmailForm />,
    attachPoint
  )
}
