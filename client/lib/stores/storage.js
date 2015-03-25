var _ = require('lodash')
var Reflux = require('reflux')


var storeActions = Reflux.createActions([
  'load',
  'set',
  'setRoom',
  'storageChange',
])
_.extend(module.exports, storeActions)

storeActions.load.sync = true

module.exports.store = Reflux.createStore({
  listenables: storeActions,

  init: function() {
    this.state = null
    this._dirtyChanges = {}
  },

  load: function() {
    if (this.state) {
      return
    }

    var data

    try {
      data = localStorage.getItem('data')
    } catch (e) {
      // localStorage is probably disabled / private browsing mode in Safari
      console.warn('unable to read localStorage')
    }

    if (data) {
      this.state = JSON.parse(data)
    } else {
      this.state = {}
    }

    if (!this.state.room) {
      this.state.room = {}
    }

    this.trigger(this.state)
  },

  storageChange: function(ev) {
    if (!this.state) {
      return
    }

    if (ev.key != 'data') {
      return
    }

    var newData = JSON.parse(ev.newValue)
    var newState = _.merge(_.assign({}, this.state, newData), this._dirtyChanges)
    if (!_.isEqual(this.state, newState)) {
      this.state = newState
      this.trigger(this.state)
    }
  },

  set: function(key, value) {
    if (_.isEqual(this.state[key], value)) {
      return
    }
    this._dirtyChanges[key] = value
    this.state[key] = value
    this.trigger(this.state)
    this._save()
  },

  setRoom: function(room, key, value) {
    if (this.state.room[room] && _.isEqual(this.state.room[room][key], value)) {
      return
    }

    if (!this._dirtyChanges.room) {
      this._dirtyChanges.room = {}
    }
    if (!this._dirtyChanges.room[room]) {
      this._dirtyChanges.room[room] = {}
    }
    this._dirtyChanges.room[room][key] = value

    if (!this.state.room[room]) {
      this.state.room[room] = {}
    }
    this.state.room[room][key] = value

    this.trigger(this.state)
    this._save()
  },

  _save: _.debounce(function() {
    var data = JSON.stringify(this.state)
    try {
      localStorage.setItem('data', data)
    } catch (e) {
      // localStorage is probably disabled / private browsing mode in Safari
      console.warn('unable to write localStorage')
    }
    this._dirtyChanges = {}
  }, 1000),
})
