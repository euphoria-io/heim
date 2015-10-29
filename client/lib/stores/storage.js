import _ from 'lodash'
import Reflux from 'reflux'


const storeActions = Reflux.createActions([
  'load',
  'set',
  'setRoom',
  'storageChange',
])
_.extend(module.exports, storeActions)

storeActions.load.sync = true

module.exports.store = Reflux.createStore({
  listenables: storeActions,

  init() {
    this.state = null
    this._dirtyChanges = {}
    this._saveThrottled = _.throttle(this._save, 1000, {leading: false})
  },

  getInitialState() {
    return this.state
  },

  load() {
    if (this.state) {
      return
    }

    let data

    try {
      data = localStorage.getItem('data')
    } catch (e) {
      // localStorage is probably disabled / private browsing mode in Safari
      console.warn('unable to read localStorage')  // eslint-disable-line no-console
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

  storageChange(ev) {
    if (!this.state) {
      return
    }

    if (ev.key !== 'data') {
      return
    }

    const newData = JSON.parse(ev.newValue)
    const newState = _.merge(_.assign({}, this.state, newData), this._dirtyChanges)
    if (!_.isEqual(this.state, newState)) {
      this.state = newState
      this.trigger(this.state)
    }
  },

  set(key, value) {
    if (_.isEqual(this.state[key], value)) {
      return
    }
    this._dirtyChanges[key] = value
    this.state[key] = value
    this.trigger(this.state)
    this._saveThrottled()
  },

  setRoom(room, key, value) {
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
    this._saveThrottled()
  },

  _save() {
    const data = JSON.stringify(this.state)
    try {
      localStorage.setItem('data', data)
    } catch (e) {
      // localStorage is probably disabled / private browsing mode in Safari
      console.warn('unable to write localStorage')  // eslint-disable-line no-console
    }
    this._dirtyChanges = {}
  },
})
