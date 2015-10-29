import _ from 'lodash'
import React from 'react'


// A button that triggers on touch start on mobile to increase responsiveness.
export default React.createClass({
  displayName: 'FastButton',

  propTypes: {
    vibrate: React.PropTypes.bool,
    disabled: React.PropTypes.bool,
    fastTouch: React.PropTypes.bool,
    empty: React.PropTypes.bool,
    onClick: React.PropTypes.func,
    component: React.PropTypes.string,
    children: React.PropTypes.node,
  },

  getDefaultProps() {
    return {
      component: 'button',
      tabIndex: 0,
    }
  },

  onClick(ev) {
    if (Heim.isTouch) {
      if (ev.type === 'touchstart') {
        if (this.props.vibrate && !this.props.disabled && Heim.isAndroid && navigator.vibrate) {
          navigator.vibrate(7)
        }

        if (this.props.fastTouch) {
          // prevent emulated click event
          ev.preventDefault()
        } else {
          return
        }
      }
    }

    if (this.props.onClick) {
      this.props.onClick(ev)
    }
  },

  onTouchStart(ev) {
    this.getDOMNode().classList.add('touching')
    this.onClick(ev)
  },

  onTouchEnd() {
    this.getDOMNode().classList.remove('touching')
  },

  onKeyDown(ev) {
    if (ev.key === 'Enter' || ev.key === 'Space') {
      this.props.onClick(ev)
    }
  },

  render() {
    // https://bugzilla.mozilla.org/show_bug.cgi?id=984869#c2
    return React.createElement(
      this.props.component,
      _.extend({}, this.props, {
        onClick: this.onClick,
        onTouchStart: this.onTouchStart,
        onTouchEnd: this.onTouchEnd,
        onTouchCancel: this.onTouchEnd,
        onKeyDown: this.onKeyDown,
      }),
      !this.props.empty && <div className="inner">{this.props.children}</div>
    )
  },
})
