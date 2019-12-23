import _ from 'lodash'
import Reflux from 'reflux'

import storage from './storage'
import actions from '../actions'


const store = module.exports.store = Reflux.createStore({
  init() {
    this.state = {
      eligible: null,
      url: process.env.HEIM_DONATION_URL || null,
    }
    this.listenTo(storage.load, this.onStorageLoad)
    this.listenTo(actions.sendMessage, this.onMessageSend)
    if (storage.store.state !== null) this.onStorageLoad()
  },

  getInitialState() {
    return this.state
  },

  onStorageLoad() {
    this.state.eligible = _.get(storage.store.state, 'sentMessage', false)
    this.trigger(this.state)
  },

  onMessageSend() {
    storage.set('sentMessage', true)
    // Will show banner on next load
  },

  _setURL(value) {
    this.state.url = value
    this.trigger(this.state)
  },
})

module.exports.openWindow = function openWindow() {
  if (store.state.url) window.open(store.state.url)
}
