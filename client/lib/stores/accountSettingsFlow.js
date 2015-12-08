import _ from 'lodash'
import Reflux from 'reflux'
import Immutable from 'immutable'

import chat from './chat'
import ImmutableMixin from './ImmutableMixin'


const storeActions = Reflux.createActions([
  'reset',
  'openSettings',
  'openChangeName',
  'openChangeEmail',
  'openChangePassword',
  'changeName',
  'changeEmail',
  'changePassword',
  'resendVerifyEmail',
  'resetPassword',
  'logout',
])
_.extend(module.exports, storeActions)

// sync so that form errors reset immediately on submit
storeActions.changeName.sync = true
storeActions.changeEmail.sync = true
storeActions.changePassword.sync = true

const StateRecord = Immutable.Record({
  email: null,
  step: 'settings',
  passwordChanged: false,
  errors: Immutable.Map(),
  working: false,
})

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {chatChange: chat.store},
    {changeNameCompleted: chat.changeName.completed},
    {changeNameFailed: chat.changeName.failed},
    {changeEmailCompleted: chat.changeEmail.completed},
    {changeEmailFailed: chat.changeEmail.failed},
    {changePasswordCompleted: chat.changePassword.completed},
    {changePasswordFailed: chat.changePassword.failed},
    {resendVerifyEmailCompleted: chat.resendVerifyEmail.completed},
    {resendVerifyEmailFailed: chat.resendVerifyEmail.failed},
    {resetPasswordCompleted: chat.resetPassword.completed},
    {resetPasswordFailed: chat.resetPassword.failed},
  ],

  mixins: [ImmutableMixin],

  init() {
    this.state = new StateRecord()
  },

  getInitialState() {
    return this.state
  },

  chatChange(state) {
    this.triggerUpdate(this.state.set('email', state.account && state.account.email))
  },

  changeNameCompleted() {
    if (this.state.get('step') === 'change-name') {
      this.triggerUpdate(new StateRecord())
    }
  },

  changeNameFailed(data) {
    this.triggerUpdate(this.state.withMutations(state => {
      const step = state.get('step')
      if (step === 'change-name') {
        const error = new Error('failed to change name: ' + data.reason)
        error.action = 'change-name'
        error.response = data
        Raven.captureException(error)
      }
    }))
  },

  changeEmailCompleted() {
    if (this.state.get('step') === 'change-email') {
      this.triggerUpdate(new StateRecord({step: 'verify-email-sent'}))
    }
  },

  changeEmailFailed(data) {
    this.triggerUpdate(this.state.withMutations(state => {
      const step = state.get('step')
      if (step === 'change-email') {
        state.set('working', false)
        if (data.error === 'access denied') {
          state.set('errors', Immutable.Map({password: 'no dice, sorry!'}))
        } else {
          const error = new Error('failed to change email: ' + data.reason)
          error.action = 'change-email'
          error.response = data
          Raven.captureException(error)
        }
      }
    }))
  },

  changePasswordCompleted() {
    if (this.state.get('step') === 'change-password') {
      this.triggerUpdate(new StateRecord({passwordChanged: true}))
    }
  },

  changePasswordFailed(data) {
    this.triggerUpdate(this.state.withMutations(state => {
      const step = state.get('step')
      if (step === 'change-password') {
        state.set('working', false)
        if (data.error === 'access denied') {
          state.set('errors', Immutable.Map({password: 'no dice, sorry!'}))
        } else {
          const error = new Error('failed to change password: ' + data.reason)
          error.action = 'change-password'
          error.response = data
          Raven.captureException(error)
        }
      }
    }))
  },

  resendVerifyEmailCompleted() {
    this.triggerUpdate(new StateRecord({step: 'verify-email-sent'}))
  },

  resendVerifyEmailFailed(data) {
    this.triggerUpdate(this.state.set('working', false))
    throw new Error('unable to resend verify email: ' + data.error)
  },

  resetPasswordCompleted() {
    this.triggerUpdate(this.state.merge({
      step: 'reset-email-sent',
      working: false,
    }))
  },

  resetPasswordFailed(data) {
    this.triggerUpdate(this.state.set('working', false))
    throw new Error('unable to reset password: ' + data.error)
  },

  reset() {
    this.triggerUpdate(new StateRecord())
  },

  openSettings() {
    this.triggerUpdate(this.state.set('step', 'settings'))
  },

  openChangeName() {
    this.triggerUpdate(this.state.set('step', 'change-name'))
  },

  openChangeEmail() {
    this.triggerUpdate(this.state.set('step', 'change-email'))
  },

  openChangePassword() {
    this.triggerUpdate(this.state.set('step', 'change-password'))
  },

  changeName(name) {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))
    chat.changeName(name)
  },

  changeEmail(email) {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))
    chat.changeEmail(email)
  },

  changePassword(oldPassword, newPassword) {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))
    chat.changePassword(oldPassword, newPassword)
  },

  resendVerifyEmail() {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))
    chat.resendVerifyEmail()
  },

  resetPassword() {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))
    chat.resetPassword(this.state.get('email'))
  },

  logout() {
    chat.logout()
  },
})
