import _ from 'lodash'
import Reflux from 'reflux'
import Immutable from 'immutable'

import heimURL from '../heim-url'
import ImmutableMixin from './immutable-mixin'
import PostFlowMixin from './post-flow-mixin'


const storeActions = Reflux.createActions([
  'initData',
  'verify',
])
_.extend(module.exports, storeActions)

storeActions.initData.sync = true

const StateRecord = Immutable.Record({
  email: null,
  confirmation: null,
  done: false,
  errors: Immutable.Map(),
  working: false,
})

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
  ],

  mixins: [
    ImmutableMixin,
    PostFlowMixin,
  ],

  init() {
    this.state = new StateRecord()
  },

  getInitialState() {
    return this.state
  },

  initData(data) {
    this.triggerUpdate(this.state.merge(data))
  },

  verify() {
    this._postAPI(heimURL('/prefs/verify'), {
      confirmation: this.state.confirmation,
    })
  },
})
