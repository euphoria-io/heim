var Reflux = require('reflux')


module.exports.store = Reflux.createStore({
  init: function() {
    this.state = {
      windowFocused: true,
      focusChangedAt: null,
    }

    uiwindow.addEventListener('focus', this.windowFocused.bind(this), false)
    uiwindow.addEventListener('blur', this.windowBlurred.bind(this), false)
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
