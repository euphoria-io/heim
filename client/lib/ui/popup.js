import React from 'react'
import ReactDOM from 'react-dom'
import classNames from 'classnames'


module.exports = React.createClass({
  displayName: 'Popup',

  propTypes: {
    className: React.PropTypes.string,
    onDismiss: React.PropTypes.func,
    children: React.PropTypes.node,
  },

  componentWillMount() {
    Heim.addEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  componentWillUnmount() {
    Heim.removeEventListener(uidocument.body, Heim.isTouch ? 'touchstart' : 'click', this.onOutsideClick, false)
  },

  onOutsideClick(ev) {
    if (!ReactDOM.findDOMNode(this).contains(ev.target) && this.props.onDismiss) {
      this.props.onDismiss(ev)
    }
  },

  render() {
    return (
      <div className={classNames('popup', this.props.className)}>
        {this.props.children}
      </div>
    )
  },
})
