import _ from 'lodash'


export default class EventListeners {
  constructor() {
    this._listeners = []
  }

  addEventListener(target, type, listener, useCapture) {
    target.addEventListener(type, listener, useCapture)
    this._listeners.push(_.toArray(arguments))
  }

  removeEventListener(target, type, listener, useCapture) {
    target.removeEventListener(type, listener, useCapture)
    const toRemove = _.toArray(arguments)
    _.remove(this._listeners, l => _.isEqual(l, toRemove))
  }

  removeAllEventListeners() {
    // iterate in reverse order so removals don't affect iteration
    _.eachRight(this._listeners, l => this.removeEventListener.apply(this, l))
  }
}
