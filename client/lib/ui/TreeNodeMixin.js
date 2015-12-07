import _ from 'lodash'


export default function(prefix) {
  const treeField = prefix ? prefix + 'Tree' : 'tree'
  const nodeField = prefix ? prefix + 'Node' : 'node'
  const nodeIdField = prefix ? prefix + 'NodeId' : 'nodeId'
  const onNodeUpdateField = prefix ? 'on' + _.capitalize(prefix) + 'NodeUpdate' : 'onNodeUpdate'

  const mixin = {
    getInitialState() {
      return {
        [nodeField]: this.props[treeField].get(this.props[nodeIdField]),
      }
    },

    componentWillReceiveProps(nextProps) {
      if (this.props[treeField] !== nextProps[treeField] || this.props[nodeIdField] !== nextProps[nodeIdField]) {
        // stop listening to old tree/node
        this.props[treeField].changes.off(this.props[nodeIdField], this[onNodeUpdateField])

        // listen to new tree/node
        nextProps[treeField].changes.on(this.props[nodeIdField], this[onNodeUpdateField])

        // update node state value
        this[onNodeUpdateField](nextProps[treeField].get(this.props[nodeIdField]))
      }
    },

    componentWillMount() {
      this.props[treeField].changes.on(this.props[nodeIdField], this[onNodeUpdateField])
    },

    componentWillUnmount() {
      this.props[treeField].changes.off(this.props[nodeIdField], this[onNodeUpdateField])
    },
  }

  mixin[onNodeUpdateField] = function handleNodeUpdateField(newValue) {
    this.setState({
      [nodeField]: newValue,
    })
  }

  return mixin
}
