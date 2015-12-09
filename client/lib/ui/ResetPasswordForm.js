import React from 'react'
import Reflux from 'reflux'

import resetPasswordFlow from '../stores/resetPasswordFlow'
import { Form, PasswordStrengthField, ErrorMessage } from './forms'
import { validatePassword, minPasswordEntropy } from './formValidators'


export default React.createClass({
  displayName: 'ResetPasswordForm',

  mixins: [
    Reflux.connect(resetPasswordFlow.store, 'flow'),
  ],

  onSubmit(values) {
    resetPasswordFlow.resetPassword(values.password)
  },

  render() {
    const flow = this.state.flow
    return (
      <Form
        ref="form"
        className="reset-password"
        onSubmit={this.onSubmit}
        validators={{
          'password': validatePassword,
        }}
        working={flow.working}
        errors={flow.errors.toJS()}
      >
        <h1>reset password</h1>
        <h2>please enter a new password for <strong>{flow.email}</strong>:</h2>
        <PasswordStrengthField
          name="password"
          label="password"
          minEntropy={minPasswordEntropy}
          showFeedback
          tabIndex={1}
        />
        <ErrorMessage name="reason" />
        {flow.done ? <button className="major-action done" disabled>your new password is saved.</button> : <button type="submit" className="major-action">save new password</button>}
      </Form>
    )
  },
})
