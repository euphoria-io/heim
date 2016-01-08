import _ from 'lodash'
import Reflux from 'reflux'

import storage from './storage'


const storeActions = Reflux.createActions([
  'windowFocused',
  'windowBlurred',
  'touch',
  'becameActive',
  'becameInactive',
])
storeActions.windowFocused.sync = true
storeActions.windowBlurred.sync = true
_.extend(module.exports, storeActions)

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {storageChange: storage.store},
  ],

  flushTime: 10 * 1000,
  idleTime: 2 * 60 * 1000,
  absenceTime: 30 * 60 * 1000,

  init() {
    this.state = {
      active: false,
      windowFocused: false,
      focusChangedAt: null,
      lastActive: {},
      lastVisit: {},
    }
    this._active = {}
    this._flushActivityThrottled = _.throttle(this._flushActivity, this.flushTime)
    this._setIdleDebounced = _.debounce(this._setInactive, this.idleTime)
  },

  getInitialState() {
    return this.state
  },

  storageChange(data) {
    if (!data) {
      return
    }
    _.each(data.room, (roomData, roomName) => {
      this.state.lastActive[roomName] = roomData.lastActive
      this.state.lastVisit[roomName] = roomData.lastVisit
    })
    this.trigger(this.state)
  },

  _flushActivity() {
    _.each(this._active, (touchTime, roomName) => {
      const lastActive = this.state.lastActive[roomName]
      if (touchTime - lastActive >= this.absenceTime) {
        storage.setRoom(roomName, 'lastVisit', lastActive)
      }
      storage.setRoom(roomName, 'lastActive', touchTime)
    })
  },

  windowFocused() {
    this.state.windowFocused = true
    this.state.focusChangedAt = Date.now()
    this.trigger(this.state)
  },

  windowBlurred() {
    this.state.windowFocused = false
    this.state.focusChangedAt = Date.now()
    this._setInactive()  // triggers
  },

  _setInactive() {
    this._setIdleDebounced.cancel()
    const wasActive = this.state.active
    this.state.active = false
    this.trigger(this.state)

    if (wasActive) {
      module.exports.becameInactive()
    }
  },

  touch(roomName) {
    const wasActive = this.state.active
    this.state.active = true
    this.trigger(this.state)
    this._setIdleDebounced()

    this._active[roomName] = Date.now()
    this._flushActivityThrottled()

    if (!wasActive) {
      module.exports.becameActive()
    }
  },
})
