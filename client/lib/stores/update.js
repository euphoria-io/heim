import _ from 'lodash'
import Reflux from 'reflux'
import Immutable from 'immutable'

import activity from './activity'
import ImmutableMixin from './ImmutableMixin'


const storeActions = Reflux.createActions([
  'prepare',
  'setReady',
  'perform',
])
_.extend(module.exports, storeActions)

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {chatChange: require('./chat').store},
    {onActive: activity.becameActive},
    {onInactive: activity.becameInactive},
  ],

  mixins: [ImmutableMixin],

  init() {
    this.state = Immutable.Map({
      ready: false,
      currentVersion: null,
      newVersion: null,
    })

    this._preparedVersion = null
    this._active = false
    this._doUpdate = null
  },

  getInitialState() {
    return this.state
  },

  chatChange(chatState) {
    const version = chatState.serverVersion
    if (!version) {
      return
    }

    const newState = this.state.withMutations(state => {
      if (!state.get('currentVersion')) {
        state.set('currentVersion', version)
      }

      if (state.get('currentVersion') !== version && state.get('newVersion') !== version) {
        state.set('newVersion', version)
        if (this._active) {
          storeActions.prepare(version)
        }
      }
    })

    this.triggerUpdate(newState)
  },

  onActive() {
    this._active = true
    storeActions.prepare(this.state.get('newVersion'))
  },

  onInactive() {
    this._active = false
  },

  prepare(version) {
    if (this._preparedVersion === version) {
      return
    }

    this._preparedVersion = version
    Heim.prepareUpdate(version)
  },

  setReady(ready, doUpdate) {
    const state = this.state.set('ready', ready)
    this._doUpdate = ready && doUpdate
    this.triggerUpdate(state)
  },

  perform() {
    this._doUpdate()
  },
})
