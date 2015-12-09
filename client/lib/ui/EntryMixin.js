import ReactDOM from 'react-dom'


export default {
  focus(withChar) {
    let node
    if (ReactDOM.findDOMNode(this).contains(uidocument.activeElement)) {
      node = uidocument.activeElement
    } else {
      node = this.refs.input
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
    this.refs.input.blur()
  },

  proxyKeyDown(ev) {
    const node = this.refs.input
    if (ev.key === 'Backspace' && ev.target !== node) {
      node.value = node.value.substr(0, node.value.length - 1)
      node.focus()
      return true
    }
    return false
  },
}
