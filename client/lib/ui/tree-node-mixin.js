var _ = require('lodash')


module.exports = function(prefix) {
  var treeField = prefix ? prefix + 'Tree' : 'tree'
  var nodeField = prefix ? prefix + 'Node' : 'node'
  var nodeIdField = prefix ? prefix + 'NodeId' : 'nodeId'
  var onNodeUpdateField = prefix ? 'on' + _.capitalize(prefix) + 'NodeUpdate' : 'onNodeUpdate'

  var mixin = {
    getInitialState: function() {
      // TODO: es6
      var state = {}
      state[nodeField] = this.props[treeField].get(this.props[nodeIdField])
      return state
    },

    componentWillReceiveProps: function(nextProps) {
      if (this.props[treeField] != nextProps[treeField] || this.props[nodeIdField] != nextProps[nodeIdField]) {
        // stop listening to old tree/node
        this.props[treeField].changes.off(this.props[nodeIdField], this[onNodeUpdateField])

        // listen to new tree/node
        nextProps[treeField].changes.on(this.props[nodeIdField], this[onNodeUpdateField])

        // update node state value
        this[onNodeUpdateField](nextProps[treeField].get(this.props[nodeIdField]))
      }
    },

    componentWillMount: function() {
      this.props[treeField].changes.on(this.props[nodeIdField], this[onNodeUpdateField])
    },

    componentWillUnmount: function() {
      this.props[treeField].changes.off(this.props[nodeIdField], this[onNodeUpdateField])
    },
  }

  mixin[onNodeUpdateField] = function(newValue) {
    // TODO: es6
    var update = {}
    update[nodeField] = newValue
    this.setState(update)
  }

  return mixin
}
