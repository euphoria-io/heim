var _ = require('lodash')
var Reflux = require('reflux')


var storeActions = Reflux.createActions([
  'windowFocused',
  'windowBlurred',
])
_.extend(module.exports, storeActions)

module.exports.store = Reflux.createStore({
  listenables: storeActions,

  init: function() {
    this.state = {
      windowFocused: false,
      focusChangedAt: null,
    }
  },

  getInitialState: function() {
    return this.state
  },

  windowFocused: function() {
    this.state.windowFocused = true
    this.state.focusChangedAt = Date.now()
    this.trigger(this.state)
  },

  windowBlurred: function() {
    this.state.windowFocused = false
    this.state.focusChangedAt = Date.now()
    this.trigger(this.state)
  },
})
