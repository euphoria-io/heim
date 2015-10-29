export default function(getter, fieldName) {
  const mixin = {
    getInitialState() {
      // TODO: es6
      const state = {}
      state[fieldName] = getter(this.props).get(this.props.nodeId)
      return state
    },

    componentWillMount() {
      getter(this.props).changes.on(this.props.nodeId, this.onDataUpdate)
    },

    componentWillUnmount() {
      getter(this.props).changes.off(this.props.nodeId, this.onDataUpdate)
    },

    onDataUpdate(newValue) {
      // TODO: es6
      const update = {}
      update[fieldName] = newValue
      this.setState(update)
    },
  }

  return mixin
}
