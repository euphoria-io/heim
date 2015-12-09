import Immutable from 'immutable'


export default {
  triggerUpdate(newState) {
    if (!Immutable.is(this.state, newState)) {
      this.state = newState
      this.trigger(this.state)
    }
  },
}
