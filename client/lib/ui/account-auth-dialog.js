import React from 'react'
import classNames from 'classnames'
import Reflux from 'reflux'

import accountAuthFlow from '../stores/account-auth-flow'
import Popup from './popup'
import FastButton from './fast-button'
import Spinner from './spinner'
import { Form, CheckField, TextField, PasswordStrengthField, ErrorMessage } from './forms'
import { validateEmail, validatePassword, validateNewPassword, minPasswordEntropy } from './form-validators'
import heimURL from '../heim-url'


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
      accountAuthFlow.signIn(values.email, values.password)
    } else if (step === 'register') {
      accountAuthFlow.register(values.email, values.newPassword.value)
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
              <button type="button" tabIndex="1" className="continue major-action" onClick={this.props.onClose}>{flow.step === 'register-email-sent' ? 'continue to account' : 'continue'}</button>
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
              {flow.showSignInButton && <button type="button" tabIndex="4" className="open-sign-in minor-action" onClick={accountAuthFlow.openSignIn}>back to sign in</button>}
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
              <button type="button" tabIndex="4" className="open-sign-in minor-secondary-action" onClick={accountAuthFlow.openSignIn}>back to sign in</button>
              <button type="submit" tabIndex="3" className="send-reminder major-secondary-action">{flow.passwordResetError || 'send a password reset email'}</button>
            </div>
          </div>
        )
      } else {
        title = 'sign in or register'
        bottom = (
          <div className="bottom">
            <div className="action-line">
              <button type="button" tabIndex="4" className={classNames('forgot', 'minor-secondary-action', flow.highlightForgot && 'highlight')} disabled={flow.working} onClick={this.onForgotClick}>forgot password?</button>
              <div className="spacer" />
              <button key="register" type="button" tabIndex="4" className="open-register minor-action" onClick={this.onRegisterClick}>register</button>
              <button key="sign-in" type="submit" tabIndex="3" className="sign-in major-action">sign in</button>
            </div>
          </div>
        )
      }

      let passwordField
      if (flow.step === 'signin') {
        passwordField = (
          <TextField
            name="password"
            label="password"
            inputType="password"
            tabIndex={2}
          />
        )
      } else if (flow.step === 'register') {
        passwordField = (
          <PasswordStrengthField
            name="newPassword"
            label="password"
            minEntropy={minPasswordEntropy}
            tabIndex={2}
          />
        )
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
            'password': flow.step === 'signin' ? validatePassword : null,
            'newPassword': flow.step === 'register' ? validateNewPassword : null,
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
      <Popup className="dialog account-auth-dialog">
        <div className="top-line">
          <div className="logo">
            <div className="emoji emoji-euphoria" />
            euphoria
          </div>
          <div className="title">{title}</div>
          <Spinner visible={flow.working} />
          <div className="spacer" />
          <FastButton className="close" onClick={this.props.onClose} />
        </div>
        {dialogContent}
      </Popup>
    )
  },
})
