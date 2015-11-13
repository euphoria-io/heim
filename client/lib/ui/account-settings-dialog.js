import React from 'react'
import Reflux from 'reflux'

import accountSettingsFlow from '../stores/account-settings-flow'
import chat from '../stores/chat'
import Dialog from './dialog'
import { Form, TextField, FieldLabelContainer, PasswordStrengthField } from './forms'
import { validateEmail, validatePassword, minPasswordEntropy } from './form-validators'
import hueHash from '../hue-hash'


export default React.createClass({
  displayName: 'AccountSettingsDialog',

  propTypes: {
    onClose: React.PropTypes.func,
  },

  mixins: [
    Reflux.connect(chat.store, 'chat'),
    Reflux.connect(accountSettingsFlow.store, 'flow'),
  ],

  getInitialState() {
    return {newAccountName: ''}
  },

  onSubmit(values) {
    const step = this.state.flow.step
    if (step === 'change-name') {
      accountSettingsFlow.changeName(values.name)
    } else if (step === 'change-email') {
      accountSettingsFlow.changeEmail(values.password.text, values.email)
    } else if (step === 'change-password') {
      accountSettingsFlow.changePassword(values.password.text, values.newPassword.text)
    }
  },

  onEditNewName(value) {
    this.setState({newAccountName: value})
  },

  openChangeName() {
    // FIXME: the form gets reset, but the state we've copied here does not :(
    // this can be fixed when this stuff is managed by store code / props
    this.setState({newAccountName: ''})
    accountSettingsFlow.openChangeName()
  },

  validateName(values, strict) {
    let error
    if (!values.name) {
      if (strict) {
        error = 'please enter a name'
      }
    } else if (values.name.length > 36) {
      error = 'that name is too long'
    }
    return {name: error}
  },

  render() {
    const flow = this.state.flow
    const account = this.state.chat.account

    let title
    let dialogContent
    const formParams = {
      key: flow.step,
      ref: 'form',
      className: 'content',
      onSubmit: this.onSubmit,
      working: flow.working,
      errors: flow.errors.toJS(),
    }
    if (flow.step === 'change-name') {
      title = 'change account name'
      dialogContent = (
        <Form
          validators={{
            'name': this.validateName,
          }}
          {...formParams}
        >
          <TextField
            name="name"
            label="new account name"
            tabIndex={1}
            onModify={this.onEditNewName}
            autoFocus
          />
          <FieldLabelContainer key="new-nick-preview" label="preview">
            <div className="field-action-box nick-preview">
              <div className="inner">
                {this.state.newAccountName.length > 0 ? <div className="big-nick" style={{background: 'hsl(' + hueHash.hue(this.state.newAccountName) + ', 65%, 85%)'}}>{this.state.newAccountName}</div> : <div className="placeholder">enter a new name</div>}
              </div>
            </div>
          </FieldLabelContainer>
          <div className="bottom">
            <div className="action-line">
              <div className="spacer" />
              <button type="button" tabIndex="3" className="minor-secondary-action" onClick={accountSettingsFlow.openSettings}>back<span className="long"> to settings</span></button>
              <button type="submit" tabIndex="2" className="register major-action">change <span className="long">account </span>name</button>
            </div>
          </div>
        </Form>
      )
    } else if (flow.step === 'change-email') {
      title = 'change email address'
      dialogContent = (
        <Form
          validators={{
            'email': validateEmail,
          }}
          {...formParams}
        >
          <TextField
            name="email"
            label="new email address"
            inputType="email"
            tabIndex={1}
            spellCheck={false}
            autoFocus
          />
          <TextField
            name="password"
            label="password"
            inputType="password"
            tabIndex={2}
          />
          <div className="bottom">
            <div className="action-line">
              <div className="spacer" />
              <button type="button" tabIndex="4" className="minor-secondary-action" onClick={accountSettingsFlow.openSettings}>back<span className="long"> to settings</span></button>
              <button type="submit" tabIndex="3" className="register major-action">change email<span className="long"> address</span></button>
            </div>
          </div>
        </Form>
      )
    } else if (flow.step === 'verify-email-sent') {
      title = 'check your email'
      dialogContent = (
        <div className="content">
          <div className="email-icon" />
          <div className="notice">ok! we've sent you a verification email.</div>
          <div className="bottom">
            <div className="action-line centered">
              <button type="button" tabIndex="1" className="continue major-action" onClick={accountSettingsFlow.openSettings}>continue</button>
            </div>
          </div>
        </div>
      )
    } else if (flow.step === 'change-password') {
      title = 'change password'
      dialogContent = (
        <Form
          validators={{
            'password': validatePassword,
            'newPassword': validatePassword,
          }}
          {...formParams}
        >
          <PasswordStrengthField
            name="password"
            label="old password"
            tabIndex={1}
            autoFocus
          />
          <PasswordStrengthField
            name="newPassword"
            label="new password"
            minEntropy={minPasswordEntropy}
            showFeedback
            tabIndex={2}
          />
          <div className="bottom">
            <div className="action-line">
              <div className="spacer" />
              <button type="button" tabIndex="4" className="minor-secondary-action" onClick={accountSettingsFlow.openSettings}>back<span className="long"> to settings</span></button>
              <button type="submit" tabIndex="3" className="register major-action">change <span className="long">account </span>password</button>
            </div>
          </div>
        </Form>
      )
    } else {
      title = 'account settings'
      dialogContent = (
        <Form {...formParams}>
          <div className="account-state">you're signed into your account. <button type="button" tabIndex="4" className="sign-out minor-secondary-action" onClick={accountSettingsFlow.logout}>sign out</button></div>
          <FieldLabelContainer label="account name">
            <div className="field-action-box">
              <div className="inner">
                <div className="big-nick" style={{background: 'hsl(' + hueHash.hue(account.get('name')) + ', 65%, 85%)'}}>{account.get('name')}</div>
              </div>
              <div className="spacer" />
              <button type="button" tabIndex="1" className="major-secondary-action" onClick={this.openChangeName}>change<span className="long"> name</span></button>
            </div>
          </FieldLabelContainer>
          <FieldLabelContainer label="email address">
            <div className="field-action-box">
              <div className="inner">{account.get('email')}</div>
              <div className="spacer" />
              <button type="button" tabIndex="2" className="major-secondary-action" onClick={accountSettingsFlow.openChangeEmail}>change<span className="long"> email</span></button>
            </div>
          </FieldLabelContainer>
          <FieldLabelContainer label="password">
            <button type="button" tabIndex="3" className="major-secondary-action" onClick={accountSettingsFlow.openChangePassword}>change <span className="long">account </span>password</button>
            {flow.passwordChanged && <span className="password-changed">saved</span>}
          </FieldLabelContainer>
        </Form>
      )
    }

    return (
      <Dialog className="account-settings-dialog" title={title} working={flow.working} onClose={this.props.onClose}>
        {dialogContent}
      </Dialog>
    )
  },
})
