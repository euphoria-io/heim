import React from 'react'
import Reflux from 'reflux'


export default React.createClass({
  displayName: 'KeyboardActionHandler',

  propTypes: {
    listenTo: React.PropTypes.func,
    keys: React.PropTypes.objectOf(React.PropTypes.func),
    children: React.PropTypes.node,
  },

  mixins: [
    Reflux.ListenerMixin,
  ],

  componentDidMount() {
    this.listenTo(this.props.listenTo, 'onKeyDown')
  },

  onKeyDown(ev) {
    let key = ev.key

    if (ev.ctrlKey) {
      key = 'Control' + key
    }

    if (ev.altKey) {
      key = 'Alt' + key
    }

    if (ev.shiftKey) {
      key = 'Shift' + key
    }

    if (ev.metaKey) {
      key = 'Meta' + key
    }

    if (key !== 'Tab' && Heim.tabPressed) {
      key = 'Tab' + key
    }

    const handler = this.props.keys[key]
    if (handler && handler(ev) !== false) {
      ev.stopPropagation()
      ev.preventDefault()
    }
  },

  render() {
    return (
      <div onKeyDown={this.onKeyDown} {...this.props}>
        {this.props.children}
      </div>
    )
  },
})
