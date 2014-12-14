var _ = require('lodash')
var Reflux = require('reflux')


var actions = Reflux.createActions([
  'set',
])
_.extend(module.exports, actions)

module.exports.store = Reflux.createStore({
  listenables: actions,

  init: function() {
    this.state = JSON.parse(localStorage.data || '{}')
  },

  getInitialState: function() {
    return this.state
  },

  set: function(key, value) {
    this.state[key] = value
    this.trigger(this.state)
    this._save()
  },

  _save: _.debounce(function() {
    localStorage.data = JSON.stringify(this.state)
  }, 1000),
})
