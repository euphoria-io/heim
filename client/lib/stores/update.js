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
    {chatChange: require('./chat').store}
  ],

  mixins: [require('./immutablemixin')],

  init: function() {
    this.state = Immutable.Map({
      ready: false,
      currentVersion: null,
      newVersion: null,
    })

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
        storeActions.prepare(version)
      }
    })

    this.triggerUpdate(state)
  },

  prepare: function(version) {
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
