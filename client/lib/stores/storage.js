var _ = require('lodash')
var Reflux = require('reflux')


var storeActions = Reflux.createActions([
  'load',
  'set',
  'setRoom',
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

    this.state = JSON.parse(localStorage.getItem('data') || '{}')

    if (!this.state.room) {
      this.state.room = {}
    }

    window.addEventListener('storage', this.onStorageUpdate, false)

    this.trigger(this.state)
  },

  onStorageUpdate: function(ev) {
    if (ev.key != 'data') {
      return
    }

    var newData = JSON.parse(ev.newValue)
    var newState = _.assign({}, this.state, newData, this._dirtyChanges)
    if (!_.isEqual(this.state, newState)) {
      this.state = newState
      this.trigger(this.state)
    }
  },

  set: function(key, value) {
    this._dirtyChanges[key] = value
    this.state[key] = value
    this.trigger(this.state)
    this._save()
  },

  setRoom: function(room, key, value) {
    var change = {room: {}}
    change.room[room] = {}
    change.room[room][key] = value

    _.merge(this._dirtyChanges, change)
    _.merge(this.state, change)
    this.trigger(this.state)
    this._save()
  },

  _save: _.debounce(function() {
    localStorage.setItem('data', JSON.stringify(this.state))
    this._dirtyChanges = {}
  }, 1000),
})
