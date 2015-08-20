var _ = require('lodash')
var Reflux = require('reflux')

var storage = require('./storage')


var storeActions = Reflux.createActions([
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

  flushTime: 60 * 1000,
  idleTime: 2 * 60 * 1000,
  absenceTime: 30 * 60 * 1000,

  init: function() {
    this.state = {
      active: false,
      windowFocused: false,
      focusChangedAt: null,
      lastActive: {},
      lastVisit: {},
    }
    this._active = {}
    this._flushActivityThrottled = _.throttle(this._flushActivity, this.flushTime, {leading: false})
    this._setIdleDebounced = _.debounce(this._setInactive, this.idleTime)
  },

  getInitialState: function() {
    return this.state
  },

  storageChange: function(data) {
    if (!data) {
      return
    }
    _.each(data.room, (roomData, roomName) => {
      this.state.lastActive[roomName] = roomData.lastActive
      this.state.lastVisit[roomName] = roomData.lastVisit
    })
    this.trigger(this.state)
  },

  _flushActivity: function() {
    _.each(this._active, (touchTime, roomName) => {
      var lastActive = this.state.lastActive[roomName]
      if (touchTime - lastActive >= this.absenceTime) {
        storage.setRoom(roomName, 'lastVisit', lastActive)
      }
      storage.setRoom(roomName, 'lastActive', touchTime)
    })
  },

  windowFocused: function() {
    this.state.windowFocused = true
    this.state.focusChangedAt = Date.now()
    this.trigger(this.state)
  },

  windowBlurred: function() {
    this.state.windowFocused = false
    this.state.focusChangedAt = Date.now()
    this._setInactive()  // triggers
  },

  _setInactive: function() {
    this._setIdleDebounced.cancel()
    var wasActive = this.state.active
    this.state.active = false
    this.trigger(this.state)

    if (wasActive) {
      module.exports.becameInactive()
    }
  },

  touch: function(roomName) {
    var wasActive = this.state.active
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
