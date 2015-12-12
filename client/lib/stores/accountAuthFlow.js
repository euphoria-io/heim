import _ from 'lodash'
import Reflux from 'reflux'
import Immutable from 'immutable'

import chat from './chat'
import ImmutableMixin from './ImmutableMixin'


const storeActions = Reflux.createActions([
  'reset',
  'openSignIn',
  'openRegister',
  'openForgot',
  'signIn',
  'register',
  'resetPassword',
])
_.extend(module.exports, storeActions)

// sync so that form errors reset immediately on submit
storeActions.signIn.sync = true
storeActions.register.sync = true
storeActions.resetPassword.sync = true

const StateRecord = Immutable.Record({
  step: 'signin',
  errors: Immutable.Map(),
  highlightForgot: false,
  showSignInButton: false,
  passwordResetError: false,
  passwordResetSent: false,
  working: false,
})

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {loginCompleted: chat.login.completed},
    {loginFailed: chat.login.failed},
    {registerCompleted: chat.registerAccount.completed},
    {registerFailed: chat.registerAccount.failed},
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

  _reloadPage() {
    // FIXME: a page navigation seems to be the most reliable way to get
    // Chrome to offer to save the password. when this is resolved, we should
    // use this.socket.reconnect().
    //
    // see: https://code.google.com/p/chromium/issues/detail?id=357696#c37

    // note: using location.replace instead of reload here so that resources
    // are loaded from cache.
    if (uiwindow.location.hash) {
      uiwindow.location.reload()
    } else {
      uiwindow.location.replace(uiwindow.location)
    }
  },

  loginCompleted() {
    if (this.state.get('step') === 'signin') {
      this._reloadPage()
    }
  },

  loginFailed(data) {
    this.triggerUpdate(this.state.withMutations(state => {
      const step = state.get('step')
      if (step === 'signin' || step === 'forgot') {
        state.set('working', false)
        if (data.reason === 'account not found') {
          state.set('errors', Immutable.Map({email: 'account not found'}))
        } else if (data.reason === 'access denied') {
          state.set('errors', Immutable.Map({password: 'no dice, sorry!'}))
          state.set('highlightForgot', true)
        } else {
          const error = new Error('failed to sign in: ' + data.reason)
          error.action = 'login'
          error.response = data
          Raven.captureException(error)
        }
      }
    }))
  },

  registerCompleted() {
    if (this.state.get('step') === 'register') {
      this.triggerUpdate(this.state.merge({
        step: 'register-email-sent',
        working: false,
      }))
    }
  },

  registerFailed(data) {
    this.triggerUpdate(this.state.withMutations(state => {
      if (state.get('step') === 'register') {
        state.set('working', false)
        if (data.reason === 'personal identity already in use') {
          state.set('errors', Immutable.Map({email: 'this email is already in use'}))
          state.set('showSignInButton', true)
        } else if (data.reason === 'not familiar yet, try again later') {
          state.set('errors', Immutable.Map({tryAgain: 'try again in a few minutes'}))
        } else {
          const error = new Error('failed to register: ' + data.reason)
          error.action = 'register'
          error.response = data
          Raven.captureException(error)
        }
      }
    }))
  },

  resetPasswordCompleted() {
    if (this.state.get('step') === 'forgot') {
      this.triggerUpdate(this.state.merge({
        step: 'reset-email-sent',
        working: false,
      }))
    }
  },

  resetPasswordFailed(data) {
    this.triggerUpdate(this.state.withMutations(state => {
      if (state.get('step') === 'forgot') {
        state.set('working', false)
        if (data.error === 'account not found') {
          state.set('errors', Immutable.Map({email: 'account not found'}))
        } else {
          state.set('passwordResetError', 'error sending. try again?')
          const error = new Error('failed to reset password: ' + data.reason)
          error.action = 'reset-password'
          error.response = data
          Raven.captureException(error)
        }
      }
    }))
  },

  reset() {
    this.triggerUpdate(new StateRecord())
  },

  openSignIn() {
    this.triggerUpdate(this.state.set('step', 'signin'))
  },

  openRegister() {
    this.triggerUpdate(this.state.set('step', 'register'))
  },

  openForgot() {
    this.triggerUpdate(this.state.set('step', 'forgot'))
  },

  signIn(email, password) {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))
    chat.login(email, password)
  },

  register(email, password) {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))
    chat.registerAccount(email, password)
  },

  resetPassword(email) {
    this.triggerUpdate(this.state.merge({
      working: true,
      errors: Immutable.Map(),
    }))
    chat.resetPassword(email)
  },
})
