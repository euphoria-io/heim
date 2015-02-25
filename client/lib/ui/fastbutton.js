var React = require('react/addons')


// A button that triggers on touch start on mobile to increase responsiveness.
module.exports = React.createClass({
  displayName: 'FastButton',

  onClick: function(ev) {
    if (Heim.isTouch && ev.type != 'touchstart') {
      return
    }
    this.props.onClick(ev)
  },

  render: function() {
    // https://bugzilla.mozilla.org/show_bug.cgi?id=984869#c2
    return (
      <button {...this.props} onClick={this.onClick} onTouchStart={this.onClick}>
        <div className="inner">{this.props.children}</div>
      </button>
    )
  },
})
