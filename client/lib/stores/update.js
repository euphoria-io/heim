var _ = require('lodash')
var Reflux = require('reflux')
var Immutable = require('immutable')


var storeActions = Reflux.createActions([
  'prepare',
  'setReady',
  'perform',
])
_.extend(module.exports, storeActions)

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {chatChange: require('./chat').store},
    {focusChange: require('./focus').store},
  ],

  mixins: [require('./immutable-mixin')],

  init: function() {
    this.state = Immutable.Map({
      ready: false,
      currentVersion: null,
      newVersion: null,
    })

    this._preparedVersion = null
    this._windowFocused = false
    this._doUpdate = null
  },

  getInitialState: function() {
    return this.state
  },

  chatChange: function(chatState) {
    var version = chatState.serverVersion
    if (!version) {
      return
    }

    var state = this.state.withMutations(state => {
      if (!state.get('currentVersion')) {
        state = state.set('currentVersion', version)
      }

      if (state.get('currentVersion') != version && state.get('newVersion') != version) {
        state = state.set('newVersion', version)
        if (this._windowFocused) {
          storeActions.prepare(version)
        }
      }
    })

    this.triggerUpdate(state)
  },

  focusChange: function(focusState) {
    this._windowFocused = focusState.windowFocused
    if (focusState.windowFocused) {
      storeActions.prepare(this.state.get('newVersion'))
    }
  },

  prepare: function(version) {
    if (this._preparedVersion == version) {
      return
    }

    this._preparedVersion = version
    Heim.prepareUpdate(version)
  },

  setReady: function(ready, doUpdate) {
    var state = this.state.set('ready', ready)
    this._doUpdate = ready && doUpdate
    this.triggerUpdate(state)
  },

  perform: function() {
    this._doUpdate()
  },
})
