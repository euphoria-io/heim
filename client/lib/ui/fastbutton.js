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
    return (
      <button {...this.props} onClick={this.onClick} onTouchStart={this.onClick} />
    )
  },
})
