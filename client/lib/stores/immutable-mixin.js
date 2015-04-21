var Immutable = require('immutable')


module.exports = {
  triggerUpdate: function(newState) {
    if (!Immutable.is(this.state, newState)) {
      this.state = newState
      this.trigger(this.state)
    }
  }
}
