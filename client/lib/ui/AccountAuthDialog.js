import React from 'react'
import classNames from 'classnames'
import Reflux from 'reflux'

import accountAuthFlow from '../stores/accountAuthFlow'
import Dialog from './dialog'
import { Form, CheckField, TextField, PasswordStrengthField, ErrorMessage } from './forms'
import { validateEmail, validatePassword, minPasswordEntropy } from './formValidators'
import heimURL from '../heimURL'


export default React.createClass({
  displayName: 'AccountAuthDialog',

  propTypes: {
    onClose: React.PropTypes.func,
  },

  mixins: [
    Reflux.connect(accountAuthFlow.store, 'flow'),
  ],

  onRegisterClick() {
    this.focusEmailIfEmpty()
    accountAuthFlow.openRegister()
  },

  onForgotClick() {
    this.focusEmailIfEmpty()
    accountAuthFlow.openForgot()
  },

  onSubmit(values) {
    const step = this.state.flow.step
    if (step === 'signin') {
      accountAuthFlow.signIn(values.email, values.password.text)
    } else if (step === 'register') {
      accountAuthFlow.register(values.email, values.password.text)
    } else if (step === 'forgot') {
      accountAuthFlow.resetPassword(values.email)
    }
  },

  focusEmailIfEmpty() {
    if (!this.state.flow.email) {
      this.refs.emailField.focus()
    }
  },

  validateAgreements(values, strict) {
    let error
    if (strict && (!values.acceptLegal || !values.acceptCommunity)) {
      error = 'please accept the agreements above'
    }
    return {agreements: error}
  },

  render() {
    const flow = this.state.flow

    let title
    let dialogContent
    if (flow.step === 'register-email-sent' || flow.step === 'reset-email-sent') {
      title = 'check your email'
      dialogContent = (
        <div className="content">
          <div className="email-icon" />
          <div className="notice">{flow.step === 'register-email-sent' ? 'done! we\'ve sent you a verification email.' : 'ok! we\'ve sent you a password reset email.'}</div>
          <div className="bottom">
            <div className="action-line centered">
              <button type="button" tabIndex="1" className="continue major-action" onClick={this.props.onClose}>continue</button>
            </div>
          </div>
        </div>
      )
    } else {
      let bottom
      if (flow.step === 'register') {
        title = 'register'
        bottom = (
          <div className="bottom green-bg">
            <div className="register-fine-print">
              hey, this is important:
              <CheckField name="acceptLegal" tabIndex={3}>
                I agree to Euphoria's <a href={heimURL('/about/terms')} target="_blank">Terms of Service</a> and <a href={heimURL('/about/privacy')} target="_blank">Privacy Policy</a>.
              </CheckField>
              <CheckField name="acceptCommunity" tabIndex={3}>
                I will respect and uphold Euphoria's <a href={heimURL('/about/conduct')} target="_blank">rules</a> and <a href={heimURL('/about/values')} target="_blank">values</a>.
              </CheckField>
            </div>
            <div className="action-line">
              <div className="spacer" />
              <ErrorMessage name="agreements" />
              <ErrorMessage name="tryAgain" />
              {flow.showSignInButton && <button type="button" tabIndex="4" className="open-sign-in minor-action" onClick={accountAuthFlow.openSignIn}>back<span className="long"> to sign in</span></button>}
              <button type="submit" tabIndex="3" className="register major-action">register</button>
            </div>
          </div>
        )
      } else if (flow.step === 'forgot') {
        title = 'forgot password?'
        bottom = (
          <div className="bottom">
            <div className="action-line">
              <div className="spacer" />
              <button type="button" tabIndex="4" className="open-sign-in minor-secondary-action" onClick={accountAuthFlow.openSignIn}>back<span className="long"> to sign in</span></button>
              <button type="submit" tabIndex="3" className="send-reminder major-secondary-action">{flow.passwordResetError || <span>send a <span className="long">password </span>reset email</span>}</button>
            </div>
          </div>
        )
      } else {
        title = 'sign in or register'
        bottom = (
          <div className="bottom">
            <div className="action-line">
              <button type="button" tabIndex="4" className={classNames('forgot', 'minor-secondary-action', flow.highlightForgot && 'highlight')} disabled={flow.working} onClick={this.onForgotClick}>forgot<span className="long"> password</span>?</button>
              <div className="spacer" />
              <button key="register" type="button" tabIndex="4" className="open-register minor-action" onClick={this.onRegisterClick}>register</button>
              <button key="sign-in" type="submit" tabIndex="3" className="sign-in major-action">sign in</button>
            </div>
          </div>
        )
      }

      let passwordField
      let passwordValidator
      if (flow.step === 'signin' || flow.step === 'register') {
        passwordField = (
          <PasswordStrengthField
            name="password"
            label="password"
            minEntropy={flow.step === 'register' ? minPasswordEntropy : null}
            showFeedback={flow.step === 'register'}
            tabIndex={2}
          />
        )
        passwordValidator = validatePassword
      }

      dialogContent = (
        <Form
          ref="form"
          className="content"
          onSubmit={this.onSubmit}
          working={flow.working}
          errors={flow.errors.toJS()}
          validators={{
            'email': validateEmail,
            'password': passwordValidator,
            'acceptLegal acceptCommunity': flow.step === 'register' ? this.validateAgreements : null,
          }}
        >
          <TextField
            ref="emailField"
            name="email"
            inputType="email"
            label="email address"
            tabIndex={1}
            spellCheck={false}
            autoFocus
          />
          {passwordField}
          {bottom}
        </Form>
      )
    }

    return (
      <Dialog className="account-auth-dialog" title={title} working={flow.working} onClose={this.props.onClose}>
        {dialogContent}
      </Dialog>
    )
  },
})
