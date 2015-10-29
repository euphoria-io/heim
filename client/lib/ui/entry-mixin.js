export default {
  focus(withChar) {
    let node
    if (this.getDOMNode().contains(uidocument.activeElement)) {
      node = uidocument.activeElement
    } else {
      node = this.refs.input.getDOMNode()
    }
    if (withChar) {
      node.value += withChar
      if (this.onChange) {
        this.onChange()
      }
    }
    node.focus()
  },

  blur() {
    this.refs.input.getDOMNode().blur()
  },

  proxyKeyDown(ev) {
    const node = this.refs.input.getDOMNode()
    if (ev.key === 'Backspace' && ev.target !== node) {
      node.value = node.value.substr(0, node.value.length - 1)
      node.focus()
      return true
    }
    return false
  },

  componentWillUnmount() {
    // FIXME: hack to work around Reflux #156.
    this.replaceState = () => {}
  },
}
