module.exports = function(getter, fieldName) {
  var mixin = {
    getInitialState: function() {
      // TODO: es6
      var state = {}
      state[fieldName] = getter(this.props).get(this.props.nodeId)
      return state
    },

    componentWillMount: function() {
      getter(this.props).changes.on(this.props.nodeId, this.onDataUpdate)
    },

    componentWillUnmount: function() {
      getter(this.props).changes.off(this.props.nodeId, this.onDataUpdate)
    },

    onDataUpdate: function(newValue) {
      // TODO: es6
      var update = {}
      update[fieldName] = newValue
      this.setState(update)
    },
  }

  return mixin
}
