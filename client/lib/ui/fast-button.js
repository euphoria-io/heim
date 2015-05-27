var _ = require('lodash')
var React = require('react')


// A button that triggers on touch start on mobile to increase responsiveness.
module.exports = React.createClass({
  displayName: 'FastButton',

  getDefaultProps: function() {
    return {component: 'button'}
  },

  onClick: function(ev) {
    if (Heim.isTouch) {
      if (ev.type == 'touchstart') {
        if (this.props.vibrate && !this.props.disabled && Heim.isAndroid && navigator.vibrate) {
          navigator.vibrate(7)
        }

        if (!this.props.fastTouch) {
          return
        }
      } else if (this.props.fastTouch) {
        return
      }
    }

    if (this.props.onClick) {
      this.props.onClick(ev)
    }
  },

  onKeyDown: function(ev) {
    if (ev.key == 'Enter' || ev.key == 'Space') {
      this.props.onClick(ev)
    }
  },

  render: function() {
    // https://bugzilla.mozilla.org/show_bug.cgi?id=984869#c2
    return React.createElement(
      this.props.component,
      _.extend({}, this.props, {
        onClick: this.onClick,
        onTouchStart: this.onClick,
        onKeyDown: this.onKeyDown,
      }),
      <div className="inner">{this.props.children}</div>
    )
  },
})
