import _ from 'lodash'
import Reflux from 'reflux'
import Immutable from 'immutable'

import chat from './chat'
import ImmutableMixin from './immutable-mixin'


const storeActions = Reflux.createActions([
  'reset',
  'openSettings',
  'openChangeName',
  'openChangeEmail',
  'openChangePassword',
  'changeName',
  'changeEmail',
  'changePassword',
  'logout',
])
_.extend(module.exports, storeActions)

// sync so that form errors reset immediately on submit
storeActions.changeName.sync = true
storeActions.changeEmail.sync = true
storeActions.changePassword.sync = true

const StateRecord = Immutable.Record({
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
  ],

  mixins: [ImmutableMixin],

  init() {
    this.state = new StateRecord()
  },

  getInitialState() {
    return this.state
  },

  changeNameCompleted() {
    if (this.state.get('step') === 'change-name') {
      this.triggerUpdate(new StateRecord())
    }
  },

  changeNameFailed() {
    this.triggerUpdate(this.state.withMutations(state => {
      const step = state.get('step')
      if (step === 'change-name') {
        // TODO
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
        }
      }
    }))
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

  logout() {
    chat.logout()
  },
})
