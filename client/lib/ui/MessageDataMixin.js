export default function(getter, fieldName) {
  const mixin = {
    getInitialState() {
      return {
        [fieldName]: getter(this.props).get(this.props.nodeId),
      }
    },

    componentWillMount() {
      getter(this.props).changes.on(this.props.nodeId, this.onDataUpdate)
    },

    componentWillUnmount() {
      getter(this.props).changes.off(this.props.nodeId, this.onDataUpdate)
    },

    onDataUpdate(newValue) {
      this.setState({
        [fieldName]: newValue,
      })
    },
  }

  return mixin
}
