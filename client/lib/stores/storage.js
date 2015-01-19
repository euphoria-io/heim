var _ = require('lodash')
var Reflux = require('reflux')


var actions = Reflux.createActions([
  'load',
  'set',
  'setRoom',
])
_.extend(module.exports, actions)

actions.load.sync = true

module.exports.store = Reflux.createStore({
  listenables: actions,

  init: function() {
    this.state = null
  },

  load: function() {
    this.state = JSON.parse(localStorage.getItem('data') || '{}')

    if (!this.state.room) {
      this.state.room = {}
    }

    this.trigger(this.state)
  },

  set: function(key, value) {
    this.state[key] = value
    this.trigger(this.state)
    this._save()
  },

  setRoom: function(room, key, value) {
    if (!this.state.room[room]) {
      this.state.room[room] = {}
    }
    this.state.room[room][key] = value
    this.trigger(this.state)
    this._save()
  },

  _save: _.debounce(function() {
    localStorage.setItem('data', JSON.stringify(this.state))
  }, 1000),
})
