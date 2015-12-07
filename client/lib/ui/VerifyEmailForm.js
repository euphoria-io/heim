import React from 'react'
import Reflux from 'reflux'

import verifyEmailFlow from '../stores/verifyEmailFlow'
import { Form, ErrorMessage } from './forms'


export default React.createClass({
  displayName: 'VerifyEmailForm',

  mixins: [
    Reflux.connect(verifyEmailFlow.store, 'flow'),
  ],

  onSubmit() {
    verifyEmailFlow.verify()
  },

  render() {
    const flow = this.state.flow
    return (
      <Form
        ref="form"
        className="verify-email"
        onSubmit={this.onSubmit}
        working={flow.working}
        errors={flow.errors.toJS()}
      >
        <h1>verify email</h1>
        <h2>shall we use <strong>{flow.email}</strong> for your euphoria account?</h2>
        <ErrorMessage name="reason" />
        {flow.done ? <button className="major-action done" disabled>great! your email address is verified.</button> : <button type="submit" tabIndex={1} className="major-action big-green-button">yep! that's me!</button>}
      </Form>
    )
  },
})
